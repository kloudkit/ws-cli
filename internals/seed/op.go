package seed

type Op string

const (
	OpCopy    Op = "copy"
	OpMerge   Op = "merge"
	OpAppend  Op = "append"
	OpPrepend Op = "prepend"
)

type SeedOp struct {
	Mode     string  `yaml:"mode"`
	Content  *string `yaml:"content"`
	Secret   bool    `yaml:"secret"`
	Op       Op      `yaml:"op"`
	Template bool    `yaml:"template"`
	Force    bool    `yaml:"force"`
}

func (o SeedOp) hasBehavior() bool {
	return o.Secret || o.Mode != "" || (o.Op != "" && o.Op != OpCopy) || o.Template
}
