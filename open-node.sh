#!/bin/sh
docker run --rm -it \
    -v "$(pwd):/src" \
    -p 3000:3000 \
    --network host \
    -w /src/webui \
    node:20-alpine \
    /bin/sh