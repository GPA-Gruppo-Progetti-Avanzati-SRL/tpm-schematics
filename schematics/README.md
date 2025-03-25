# README

## Regions

```mermaid
---
title: Region state machine
---
stateDiagram-v2
[*] --> OutOfRegion
OutOfRegion --> StartedRegion: on start region
StartedRegion --> OutOfRegion: on end region name coincident
```

Nella fase di merging si applicano le seguenti modalità.
Nella colonna current la condizione sul documento esistente mentre sulla colonna new il documento in fase di generazione.

| new                    | current                          | behaviour                      |
|------------------------|----------------------------------|--------------------------------|
| the region is empty    | the region is not present        | use the new region content     |
| the region is empty    | the region exists and empty      | use the new region content     |
| the region is empty    | the region exists with content   | use the current region content |
| the region has content | the region is not present        | use the new region content     |
| the region has content | the region is  present and empty | use the new region content     |
| the region has content | the region exists with content   | keep the old content           |

Bottom line: if the current region exists and has content use that content, otherwise use the new content (empty or not).

```mermaid
---
title: Merging Region state machine (without demarcation logic)
---
stateDiagram-v2
[*] --> OutOfRegion
OutOfRegion --> StartedRegion: on start region
StartedRegion --> InRegionSkipContent: the fetched region does exists
InRegionSkipContent --> OutOfRegion: exit region on end
StartedRegion --> InRegion: the fetched region does not exists
InRegion --> OutOfRegion: exit region on end
```
