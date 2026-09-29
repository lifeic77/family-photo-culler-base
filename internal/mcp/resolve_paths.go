package mcp

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"photofield/internal/collection"
	"photofield/internal/image"
)

const maxResolvePhotoPaths = 100

type resolvePhotoPathsInput struct {
	CollectionId string   `json:"collection_id" jsonschema:"The collection ID from list_collections."`
	Paths        []string `json:"paths" jsonschema:"Absolute photo paths to resolve to Photofield file IDs. Maximum 100."`
}

type resolvedPhotoPath struct {
	Path             string `json:"path"`
	FileId           int    `json:"file_id"`
	Width            int    `json:"width,omitempty"`
	Height           int    `json:"height,omitempty"`
	CreatedAt        string `json:"created_at,omitempty"`
	PreviewUrl       string `json:"preview_url,omitempty"`
	CachedPreviewUrl string `json:"cached_preview_url,omitempty"`
}

type resolvePhotoPathsOutput struct {
	Items   []resolvedPhotoPath `json:"items"`
	Missing []string            `json:"missing,omitempty"`
}

func applyResolvedPhotoInfo(item *resolvedPhotoPath, info image.Info, serverBaseURL, apiPrefix string) {
	item.Width = info.Width
	item.Height = info.Height
	if !info.DateTime.IsZero() {
		item.CreatedAt = info.DateTime.Format("2006-01-02T15:04:05Z07:00")
	}
	if serverBaseURL != "" && item.Path != "" {
		filename := filepath.Base(item.Path)
		previewFilename := strings.TrimSuffix(filename, filepath.Ext(filename)) + "_preview.jpg"
		item.PreviewUrl = fileURL(
			serverBaseURL,
			apiPrefix,
			fmt.Sprintf("/files/%d/previews/%s?w=400", item.FileId, previewFilename),
		)
		item.CachedPreviewUrl = fileURL(
			serverBaseURL,
			apiPrefix,
			fmt.Sprintf("/files/%d/previews/%s?cache_only=true", item.FileId, previewFilename),
		)
	}
}

func findCollection(collections *[]collection.Collection, id string) *collection.Collection {
	for i := range *collections {
		if (*collections)[i].Id == id {
			return &(*collections)[i]
		}
	}
	return nil
}

func canonicalCollectionPath(raw string, dirs []string) (string, error) {
	if raw == "" || !filepath.IsAbs(raw) {
		return "", fmt.Errorf("absolute path required")
	}
	resolved, err := filepath.EvalSymlinks(raw)
	if err != nil {
		return "", fmt.Errorf("unable to resolve path: %w", err)
	}
	resolved, err = filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("unable to normalize path: %w", err)
	}
	resolved = filepath.Clean(resolved)

	for _, dir := range dirs {
		dirResolved, err := filepath.EvalSymlinks(filepath.Clean(dir))
		if err != nil {
			continue
		}
		dirResolved, err = filepath.Abs(dirResolved)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(filepath.Clean(dirResolved), resolved)
		if err != nil {
			continue
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return resolved, nil
		}
	}
	return "", fmt.Errorf("path is outside collection directories")
}

func resolvePhotoPathsHandler(collections *[]collection.Collection, imageSource *image.Source, srv *Server) mcp.ToolHandlerFor[resolvePhotoPathsInput, resolvePhotoPathsOutput] {
	return func(_ context.Context, _ *mcp.CallToolRequest, input resolvePhotoPathsInput) (*mcp.CallToolResult, resolvePhotoPathsOutput, error) {
		coll := findCollection(collections, input.CollectionId)
		if coll == nil {
			return nil, resolvePhotoPathsOutput{}, fmt.Errorf("collection not found: %s", input.CollectionId)
		}
		if len(input.Paths) == 0 {
			return nil, resolvePhotoPathsOutput{}, fmt.Errorf("at least one path is required")
		}
		if len(input.Paths) > maxResolvePhotoPaths {
			return nil, resolvePhotoPathsOutput{}, fmt.Errorf("too many paths: maximum is %d", maxResolvePhotoPaths)
		}

		ordered := make([]string, 0, len(input.Paths))
		targets := make(map[string]struct{}, len(input.Paths))
		aliases := make(map[string]string, len(input.Paths)*2)
		for _, raw := range input.Paths {
			canonical, err := canonicalCollectionPath(raw, coll.Dirs)
			if err != nil {
				return nil, resolvePhotoPathsOutput{}, fmt.Errorf("invalid path %q: %w", raw, err)
			}
			rawAbs, err := filepath.Abs(filepath.Clean(raw))
			if err != nil {
				return nil, resolvePhotoPathsOutput{}, fmt.Errorf("unable to normalize path %q: %w", raw, err)
			}
			if _, exists := targets[canonical]; !exists {
				targets[canonical] = struct{}{}
				ordered = append(ordered, canonical)
			}
			aliases[filepath.Clean(rawAbs)] = canonical
			aliases[filepath.Clean(canonical)] = canonical
		}

		// Use Photofield's existing SQLite id/path stream instead of resolving every
		// indexed file through the filesystem. Only the requested <=100 paths above
		// touch the filesystem for canonical/symlink validation.
		found := make(map[string]int, len(targets))
		for idPath := range imageSource.ListIdPaths(coll.Dirs, 0) {
			indexed, err := filepath.Abs(filepath.Clean(idPath.Path))
			if err != nil {
				continue
			}
			canonical, wanted := aliases[filepath.Clean(indexed)]
			if !wanted {
				continue
			}
			found[canonical] = int(idPath.Id)
			if len(found) == len(targets) {
				break
			}
		}

		out := resolvePhotoPathsOutput{Items: make([]resolvedPhotoPath, 0, len(found))}
		serverBaseURL := ""
		apiPrefix := ""
		if srv != nil {
			apiPrefix = srv.apiPrefix
			if value := srv.baseURL.Load(); value != nil {
				serverBaseURL, _ = value.(string)
			}
		}
		for _, path := range ordered {
			if id, ok := found[path]; ok {
				item := resolvedPhotoPath{Path: path, FileId: id}
				info, _ := imageSource.GetCachedInfo(image.ImageId(id))
				applyResolvedPhotoInfo(&item, info, serverBaseURL, apiPrefix)
				out.Items = append(out.Items, item)
			} else {
				out.Missing = append(out.Missing, path)
			}
		}
		return nil, out, nil
	}
}
