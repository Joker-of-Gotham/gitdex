package gitops

import (
	"context"
	"strconv"
	"strings"
	"time"
)

type LogOptions struct {
	MaxCount    int
	Since       string
	Until       string
	Author      string
	Grep        string
	Paths       []string
	FirstParent bool
	Oneline     bool
	Format      string
}

type LogEntry struct {
	SHA         string
	ShortSHA    string
	Author      string
	AuthorEmail string
	Date        time.Time
	Subject     string
	Body        string
	Parents     []string
}

type ObjectInfo struct {
	Type    string
	Size    int64
	Content string
}

type BlameOptions struct {
	StartLine int
	EndLine   int
}

type BlameLine struct {
	SHA     string
	Author  string
	Date    time.Time
	LineNo  int
	Content string
}

type DiskUsage struct {
	Count    int
	Size     string
	InPack   int
	PackSize string
	Prunable int
	Garbage  int
}

type LsFilesOptions struct {
	Others  bool
	Ignored bool
	Cached  bool
}

type TreeEntry struct {
	Mode string
	Type string
	SHA  string
	Path string
}

type RefEntry struct {
	SHA  string
	Name string
	Type string
}

type HistoryInspector struct {
	executor *GitExecutor
}

func NewHistoryInspector(executor *GitExecutor) *HistoryInspector {
	return &HistoryInspector{executor: executor}
}

func (hi *HistoryInspector) Log(ctx context.Context, repoPath string, opts *LogOptions) ([]LogEntry, error) {
	format := "%H%x00%h%x00%an%x00%ae%x00%aI%x00%s%x00%b%x00%P"
	if opts != nil && opts.Format != "" {
		format = opts.Format
	}
	args := []string{"log", "--format=" + format}
	if opts != nil {
		if opts.MaxCount > 0 {
			args = append(args, "-n", strconv.Itoa(opts.MaxCount))
		}
		if opts.Since != "" {
			args = append(args, "--since", opts.Since)
		}
		if opts.Until != "" {
			args = append(args, "--until", opts.Until)
		}
		if opts.Author != "" {
			args = append(args, "--author", opts.Author)
		}
		if opts.Grep != "" {
			args = append(args, "--grep", opts.Grep)
		}
		if opts.FirstParent {
			args = append(args, "--first-parent")
		}
		if opts.Oneline {
			args = append(args, "--oneline")
		}
	}
	if opts != nil && len(opts.Paths) > 0 {
		args = append(args, "--")
		args = append(args, opts.Paths...)
	}
	lines, err := hi.executor.RunLines(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	return hi.parseLogEntries(lines, format), nil
}

func (hi *HistoryInspector) LogBetween(ctx context.Context, repoPath string, from string, to string, opts *LogOptions) ([]LogEntry, error) {
	format := "%H%x00%h%x00%an%x00%ae%x00%aI%x00%s%x00%b%x00%P"
	if opts != nil && opts.Format != "" {
		format = opts.Format
	}
	args := []string{"log", "--format=" + format}
	if opts != nil {
		if opts.MaxCount > 0 {
			args = append(args, "-n", strconv.Itoa(opts.MaxCount))
		}
		if opts.FirstParent {
			args = append(args, "--first-parent")
		}
		if opts.Oneline {
			args = append(args, "--oneline")
		}
	}
	args = append(args, from+".."+to)
	if opts != nil && len(opts.Paths) > 0 {
		args = append(args, "--")
		args = append(args, opts.Paths...)
	}
	lines, err := hi.executor.RunLines(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	return hi.parseLogEntries(lines, format), nil
}

func (hi *HistoryInspector) parseLogEntries(lines []string, format string) []LogEntry {
	var entries []LogEntry
	var current *LogEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.Contains(format, "%x00") {
			parts := strings.Split(line, "\x00")
			e := LogEntry{}
			if len(parts) >= 1 {
				e.SHA = parts[0]
			}
			if len(parts) >= 2 {
				e.ShortSHA = parts[1]
			}
			if len(parts) >= 3 {
				e.Author = parts[2]
			}
			if len(parts) >= 4 {
				e.AuthorEmail = parts[3]
			}
			if len(parts) >= 5 {
				if t, err := time.Parse(time.RFC3339, parts[4]); err == nil {
					e.Date = t
				}
			}
			if len(parts) >= 6 {
				e.Subject = parts[5]
			}
			if len(parts) >= 7 {
				e.Body = strings.TrimSpace(parts[6])
			}
			if len(parts) >= 8 && parts[7] != "" {
				e.Parents = strings.Fields(parts[7])
			}
			entries = append(entries, e)
		} else {
			if current == nil {
				current = &LogEntry{}
			}
			current.Subject = line
			entries = append(entries, *current)
			current = nil
		}
	}
	return entries
}

func (hi *HistoryInspector) Show(ctx context.Context, repoPath string, revision string) (string, error) {
	args := []string{"show"}
	if revision != "" {
		args = append(args, revision)
	}
	result, err := hi.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func (hi *HistoryInspector) ShowFile(ctx context.Context, repoPath string, revision string, path string) (string, error) {
	args := []string{"show"}
	if revision != "" {
		args = append(args, revision+":"+path)
	} else {
		args = append(args, path)
	}
	result, err := hi.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func (hi *HistoryInspector) Describe(ctx context.Context, repoPath string, revision string, tags bool) (string, error) {
	args := []string{"describe"}
	if tags {
		args = append(args, "--tags")
	}
	if revision != "" {
		args = append(args, revision)
	}
	result, err := hi.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (hi *HistoryInspector) Shortlog(ctx context.Context, repoPath string, opts *LogOptions) (string, error) {
	args := []string{"shortlog"}
	if opts != nil {
		if opts.Since != "" {
			args = append(args, "--since", opts.Since)
		}
		if opts.Until != "" {
			args = append(args, "--until", opts.Until)
		}
	}
	if opts != nil && len(opts.Paths) > 0 {
		args = append(args, "--")
		args = append(args, opts.Paths...)
	}
	result, err := hi.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return result.Stdout, nil
}

func (hi *HistoryInspector) Blame(ctx context.Context, repoPath string, path string, opts *BlameOptions) ([]BlameLine, error) {
	args := []string{"blame", "-p"}
	if opts != nil {
		if opts.StartLine > 0 && opts.EndLine > 0 {
			args = append(args, "-L", strconv.Itoa(opts.StartLine)+","+strconv.Itoa(opts.EndLine))
		}
	}
	args = append(args, path)
	result, err := hi.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	return hi.parseBlame(result.Stdout), nil
}

func (hi *HistoryInspector) parseBlame(output string) []BlameLine {
	var lines []BlameLine
	var current *BlameLine
	lineNo := 0
	for _, rawLine := range strings.Split(output, "\n") {
		if strings.HasPrefix(rawLine, "\t") {
			lineNo++
			if current != nil {
				current.LineNo = lineNo
				current.Content = strings.TrimPrefix(rawLine, "\t")
				lines = append(lines, *current)
			}
			current = nil
			continue
		}
		if len(rawLine) >= 40 && rawLine[40] == ' ' {
			if current != nil {
				lines = append(lines, *current)
			}
			current = &BlameLine{
				SHA: strings.TrimSpace(rawLine[:40]),
			}
		} else if current != nil {
			if strings.HasPrefix(rawLine, "author ") {
				current.Author = strings.TrimSpace(strings.TrimPrefix(rawLine, "author "))
			} else if strings.HasPrefix(rawLine, "author-time ") {
				if ts, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(rawLine, "author-time ")), 10, 64); err == nil {
					current.Date = time.Unix(ts, 0)
				}
			}
		}
	}
	if current != nil {
		lines = append(lines, *current)
	}
	return lines
}

func (hi *HistoryInspector) LsFiles(ctx context.Context, repoPath string, opts *LsFilesOptions) ([]string, error) {
	args := []string{"ls-files"}
	if opts != nil {
		if opts.Others {
			args = append(args, "--others")
		}
		if opts.Ignored {
			args = append(args, "--ignored")
		}
		if opts.Cached {
			args = append(args, "--cached")
		}
	}
	return hi.executor.RunLines(ctx, repoPath, args...)
}

func (hi *HistoryInspector) LsTree(ctx context.Context, repoPath string, treeish string, recursive bool) ([]TreeEntry, error) {
	args := []string{"ls-tree"}
	if recursive {
		args = append(args, "-r")
	}
	args = append(args, treeish)
	result, err := hi.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	return hi.parseLsTree(result.Stdout), nil
}

func (hi *HistoryInspector) parseLsTree(stdout string) []TreeEntry {
	var entries []TreeEntry
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			e := TreeEntry{
				Mode: fields[0],
				Type: fields[1],
				SHA:  fields[2],
				Path: strings.Join(fields[3:], " "),
			}
			entries = append(entries, e)
		}
	}
	return entries
}

func (hi *HistoryInspector) CatFile(ctx context.Context, repoPath string, object string, typ string) (*ObjectInfo, error) {
	tResult, err := hi.executor.Run(ctx, repoPath, "cat-file", "-t", object)
	if err != nil {
		return nil, err
	}
	objType := strings.TrimSpace(tResult.Stdout)
	oi := &ObjectInfo{Type: objType}

	sResult, err := hi.executor.Run(ctx, repoPath, "cat-file", "-s", object)
	if err == nil {
		if sz, err := strconv.ParseInt(strings.TrimSpace(sResult.Stdout), 10, 64); err == nil {
			oi.Size = sz
		}
	}

	contentType := objType
	if typ != "" {
		contentType = typ
	}
	contentResult, err := hi.executor.Run(ctx, repoPath, "cat-file", contentType, object)
	if err == nil {
		oi.Content = contentResult.Stdout
	}
	return oi, nil
}

func (hi *HistoryInspector) ForEachRef(ctx context.Context, repoPath string, pattern string, format string) ([]RefEntry, error) {
	args := []string{"for-each-ref"}
	if format != "" {
		args = append(args, "--format="+format)
	} else {
		args = append(args, "--format=%(objectname)%x00%(refname:short)%x00%(objecttype)")
	}
	if pattern != "" {
		args = append(args, pattern)
	}
	lines, err := hi.executor.RunLines(ctx, repoPath, args...)
	if err != nil {
		return nil, err
	}
	var entries []RefEntry
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\x00", 3)
		e := RefEntry{}
		if len(parts) >= 1 {
			e.SHA = parts[0]
		}
		if len(parts) >= 2 {
			e.Name = parts[1]
		}
		if len(parts) >= 3 {
			e.Type = parts[2]
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func (hi *HistoryInspector) ShowRef(ctx context.Context, repoPath string, ref string) (string, error) {
	args := []string{"show-ref", ref}
	result, err := hi.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (hi *HistoryInspector) RevParse(ctx context.Context, repoPath string, revision string) (string, error) {
	args := []string{"rev-parse", revision}
	result, err := hi.executor.Run(ctx, repoPath, args...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(result.Stdout), nil
}

func (hi *HistoryInspector) CountObjects(ctx context.Context, repoPath string) (*DiskUsage, error) {
	result, err := hi.executor.Run(ctx, repoPath, "count-objects", "-v")
	if err != nil {
		return nil, err
	}
	return hi.parseCountObjects(result.Stdout), nil
}

func (hi *HistoryInspector) parseCountObjects(stdout string) *DiskUsage {
	du := &DiskUsage{}
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "count: ") {
			du.Count, _ = strconv.Atoi(strings.TrimPrefix(line, "count: "))
		} else if strings.HasPrefix(line, "size: ") {
			du.Size = strings.TrimPrefix(line, "size: ")
		} else if strings.HasPrefix(line, "in-pack: ") {
			du.InPack, _ = strconv.Atoi(strings.TrimPrefix(line, "in-pack: "))
		} else if strings.HasPrefix(line, "size-pack: ") {
			du.PackSize = strings.TrimPrefix(line, "size-pack: ")
		} else if strings.HasPrefix(line, "prune-packable: ") {
			du.Prunable, _ = strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "prune-packable: ")))
		} else if strings.HasPrefix(line, "garbage: ") {
			du.Garbage, _ = strconv.Atoi(strings.TrimPrefix(line, "garbage: "))
		}
	}
	return du
}
