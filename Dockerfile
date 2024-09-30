# Copyright (c) 2024, Intel Corporation.  All rights reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

ARG VERSION=1.16.0

# Stage 0: Use the BASE_IMAGE as a named stage to extract Habana libraries
ARG BASE_IMAGE
FROM ${BASE_IMAGE} as habana_tools_base

# Use official Golang image for building the binaries
FROM golang:1.21.5 AS builder

# Install additional dependencies for building
RUN apt-get update && \
    apt-get install -y wget make git gcc && \
    rm -rf /var/lib/apt/lists/*

# Set up Go environment
ENV GOPATH /go
ENV PATH $GOPATH/bin:/usr/local/go/bin:$PATH

WORKDIR /go/src/habanalabs-device-plugin

# Copy Habana libraries required for building
COPY --from=habana_tools_base /usr/lib/habanalabs /usr/lib/habanalabs

# Copy source code
COPY . .

# Clean up Go module dependencies
RUN go mod tidy

# Build the normal binary
RUN go build -buildvcs=false -o bin/habanalabs-device-plugin .

# Build the fake version with `-tags=fakehlml`
RUN go build -tags=fakehlml -buildvcs=false -o bin/habanalabs-device-plugin-fake .

# Create the final runtime image using minimal `ubuntu` base image
FROM ubuntu:22.04

# Install necessary runtime dependencies
RUN apt update && apt install -y --no-install-recommends \
    pciutils && \
    rm -rf /var/lib/apt/lists/*

# Copy the built binaries from the builder stage
COPY --from=builder /go/src/habanalabs-device-plugin/bin/habanalabs-device-plugin /usr/bin/habanalabs-device-plugin
COPY --from=builder /go/src/habanalabs-device-plugin/bin/habanalabs-device-plugin-fake /usr/bin/habanalabs-device-plugin-fake
# Copy Habana libraries from the base image stage (habana_base)
COPY --from=habana_tools_base /usr/lib/habanalabs/libhlml.so /usr/lib/habanalabs/libhlml.so

# Copy Habana smi tool from the base image stage (habana_base)
COPY --from=habana_tools_base /usr/bin/hl-smi /usr/bin/hl-smi

# Configure dynamic linker run-time bindings
RUN echo "/usr/lib/habanalabs/" >> /etc/ld.so.conf.d/habanalabs.conf && ldconfig

# Add metadata to the image
ARG BUILD_DATE
ARG BUILD_REF

LABEL   io.k8s.display-name="HABANA Device Plugin" \
        vendor="HABANA LABS" \
        version=${VERSION} \
        image.git-commit="${GIT_COMMIT}" \
        image.created="${BUILD_DATE}" \
        image.revision="${BUILD_REF}" \
        summary="HABANA device plugin for Kubernetes" \
		description="See summary"

# Copy the shell script into the image
COPY entrypoint.sh /usr/bin/entrypoint.sh

# Set executable permissions on the script
RUN chmod +x /usr/bin/entrypoint.sh

# Set the entrypoint to use the script
ENTRYPOINT ["/usr/bin/entrypoint.sh"]
