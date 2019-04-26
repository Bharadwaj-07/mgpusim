# L1V Cache

## Overview

L1V cache is the customized L1 cache for the GCN3 architecture.

## Timing

The "Coalesce Stage" extracts requests from the `TopPort`. If this is the first
request or this request is accessing the same block as the previous requests,
the coalescer buffers the request. If the request cannot coalesce with buffered
requests, the Coalesce Stage send the all the requests in the buffer as a
bundle to the directory stage. The Coalesce Stage also clears the buffer and
put the new request in the buffer. In the case that the request is the last
request from a wave instruction, the Coalesce Stage immediately performs
coalescing.

The "Directory Stage" considers 6 cases:

1. Read MSHR hit:

    In this case, the read transaction is attached to the MSHR entry. Nothing
    else should happen.

1. Read hit:

1. Read miss:

1. Write MSHR hit:

1. Write hit:

1. Write miss: