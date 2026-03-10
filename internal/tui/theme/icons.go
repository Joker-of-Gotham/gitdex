package theme

type IconSet struct {
	Added     string
	Modified  string
	Deleted   string
	Untracked string
	Staged    string
	Branch    string
	Stash     string
	Warning   string
	Error     string
	Success   string
	Info      string
}

var Icons IconSet

func InitIcons() {
	Icons = IconSet{
		Added:     "+",
		Modified:  "~",
		Deleted:   "-",
		Untracked: "?",
		Staged:    "*",
		Branch:    ">",
		Stash:     "#",
		Warning:   "!",
		Error:     "x",
		Success:   "v",
		Info:      "i",
	}
}
