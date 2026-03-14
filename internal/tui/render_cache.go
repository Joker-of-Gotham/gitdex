package tui

import (
	"fmt"
	"hash/fnv"
	"strconv"
	"strings"
)

type renderCache struct {
	text    map[string]renderTextCacheEntry
	regions map[string]renderRegionsCacheEntry
}

type renderTextCacheEntry struct {
	key     string
	content string
}

type renderRegionsCacheEntry struct {
	key     string
	content string
	clicks  []clickRegion
}

func newRenderCache() *renderCache {
	return &renderCache{
		text:    make(map[string]renderTextCacheEntry),
		regions: make(map[string]renderRegionsCacheEntry),
	}
}

func (c *renderCache) textValue(section, key string) (string, bool) {
	if c == nil {
		return "", false
	}
	entry, ok := c.text[section]
	if !ok || entry.key != key {
		return "", false
	}
	return entry.content, true
}

func (c *renderCache) setTextValue(section, key, content string) {
	if c == nil {
		return
	}
	c.text[section] = renderTextCacheEntry{key: key, content: content}
}

func (c *renderCache) regionValue(section, key string) (string, []clickRegion, bool) {
	if c == nil {
		return "", nil, false
	}
	entry, ok := c.regions[section]
	if !ok || entry.key != key {
		return "", nil, false
	}
	clicks := append([]clickRegion(nil), entry.clicks...)
	return entry.content, clicks, true
}

func (c *renderCache) setRegionValue(section, key, content string, clicks []clickRegion) {
	if c == nil {
		return
	}
	c.regions[section] = renderRegionsCacheEntry{
		key:     key,
		content: content,
		clicks:  append([]clickRegion(nil), clicks...),
	}
}

func renderHash(parts ...string) string {
	hasher := fnv.New64a()
	for _, part := range parts {
		_, _ = hasher.Write([]byte(part))
		_, _ = hasher.Write([]byte{0})
	}
	return fmt.Sprintf("%x", hasher.Sum64())
}

func renderHashInts(parts ...int) string {
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		items = append(items, strconv.Itoa(part))
	}
	return renderHash(items...)
}

func joinHashParts(parts ...string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		clean = append(clean, strings.TrimSpace(part))
	}
	return renderHash(clean...)
}
