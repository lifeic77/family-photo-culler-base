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

## Current culling controls

The `family-culler` branch adds a minimal control strip to the single-photo viewer. It submits only `collection_id`, `file_id`, and the selected action to the sidecar; original-path resolution stays server-side.

Actions and shortcuts:

- `1` — keeper (`KEEP_ONE`)
- `2` — keep but do not select (`KEEP_ARCHIVE`)
- `3` — reject / delete candidate (`REJECT_ONE`)
- `4` — intentional-motion protection (`INTENTIONAL_MOTION`)

The current action is fetched from the sidecar and highlighted in the Photofield controls. Arrow-key navigation remains Photofield-native. No culling action directly deletes or moves a file.

The browser sidecar endpoint defaults to `http://127.0.0.1:8767` and can be overridden at runtime with `localStorage.fpcSidecarUrl`. The sidecar separately validates the exact Photofield browser Origin.

## Duplicate review bridge

The `family-culler` branch also exposes a narrow MCP tool for local analysis sidecars:

```text
resolve_photo_paths(collection_id, paths[])
```

- maximum 100 absolute paths per call
- every requested path is canonicalized and must remain inside the requested collection directories
- lookup uses Photofield's existing SQLite `ListIdPaths` stream, so resolving candidates does not walk/read every original file on NAS
- the tool is read-only and does not scan, tag, move, or delete files

The sidecar uses this only to map Czkawka filesystem results back to Photofield IDs. Duplicate scans are started from the local CLI, never from the browser. When a duplicate manifest exists, the single-photo viewer shows the strongest matching group (exact before similar) and offers two mouse-only decisions:

- **不是重复** (`NOT_DUPLICATE`)
- **确认重复** (`DUPLICATE_CONFIRMED`)

`DUPLICATE_CONFIRMED` has a confirmation dialog and only records human delete intent. It still requires the separate delete-plan → approval → quarantine workflow. The browser cannot supply the trusted group membership; the sidecar reloads the Czkawka manifest and resolves the group server-side.
