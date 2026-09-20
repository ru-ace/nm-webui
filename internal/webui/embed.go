package webui

import "embed"

// Dist holds the built Svelte SPA. It is populated by `npm run build`
// (vite outDir = ../../internal/webui/dist) before `go build`; a placeholder
// is committed so the binary always compiles.
//
//go:embed dist
var Dist embed.FS

// DistDir is the in-embed path containing index.html and assets.
const DistDir = "dist"