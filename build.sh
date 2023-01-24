#!/usr/bin/bash
# Copyright (C) 2021 - 2023 iDigitalFlame
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http//#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

output="$(pwd)/bin/huectl"
if [ $# -ge 1 ]; then
    output="$1"
fi

echo "Building.."
go build -buildvcs=false -trimpath -ldflags "-s -w -X main.version=$(date +%F)_$(git rev-parse --short HEAD 2> /dev/null || echo "non-git")" -o "$output" cmd/main.go

which upx &> /dev/null
if [ $? -eq 0 ] && [ -f "$output" ]; then
    upx --compress-exports=1 --strip-relocs=1 --compress-icons=2 --best --no-backup -9 "$output"
fi

echo "Done!"
