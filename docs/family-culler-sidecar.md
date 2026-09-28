# Family Photo Culler Sidecar Integration

This fork keeps Photofield as the photo-library base and integrates the separate `family-photo-culler` project as a local culling sidecar.

## Branch policy

- `main`: stay close to `SmilyOrg/photofield:main`.
- `family-culler`: carry the smallest possible UI/integration delta.
- Prefer upstream APIs and MCP tools over changes to Photofield's cache schema or indexer.

## Division of responsibility

Photofield owns:

- read-only filesystem collection discovery
- incremental SQLite cache/index state
- EXIF metadata and thumbnail pipelines
- timeline/gallery rendering
- file IDs, search, tags, face/search metadata
- `/mcp` tools such as `list_collections`, `search_photos`, `get_photo`, and `get_photo_metadata`

The sidecar owns:

- family/street culling semantics
- burst keeper/reject feedback
- optional Luna review
- preference learning
- explicit delete intent
- frozen delete plans, approval, recoverable quarantine, and restore

## Identity contract

Photofield `file_id` is the UI identity. The sidecar resolves it through the standard Photofield MCP tool:

```text
get_photo_metadata(file_id)
```

The result includes the original absolute `path`, dimensions, preview URL, and original URL. The sidecar validates returned paths against configured allowed photo roots before accepting feedback or building a delete plan.

No Photofield file ID, tag, search result, similarity score, or AI output is deletion authority.

## Source safety

Original photo collections should be mounted/read as read-only. Cache/database data belongs on the separate work volume. Any future culling UI controls in this fork must call the sidecar and must not add a shortcut that writes, moves, or deletes originals directly through Photofield.

## First integration target

Add minimal culling controls to the Vue photo viewer/grid:

- keeper
- archive-but-not-selected
- reject/delete-candidate
- intentional-motion protection

The UI should submit only `collection_id`, `file_id`, action, and optional group context. Original-path resolution stays server-side in the sidecar.
