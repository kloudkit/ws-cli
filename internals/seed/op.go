package seed

type Op string

const (
	OpCopy    Op = "copy"
	OpMerge   Op = "merge"
	OpAppend     Op = "append"
	OpPrepend    Op = "prepend"
	OpBlock      Op = "block"
	OpLineInfile Op = "lineinfile"
)

type SeedOp struct {
	Mode     string  `yaml:"mode"`
	Content  *string `yaml:"content"`
	Secret   bool    `yaml:"secret"`
	Op       Op      `yaml:"op"`
	Template bool    `yaml:"template"`
	Force    bool    `yaml:"force"`
	Comment  string  `yaml:"comment"`
}

func (o SeedOp) hasBehavior() bool {
	return o.Secret || o.Mode != "" || (o.Op != "" && o.Op != OpCopy) || o.Template || o.Content != nil
}

func (o Op) inPlace() bool {
	return o == OpMerge || o == OpAppend || o == OpPrepend || o == OpBlock || o == OpLineInfile
}
