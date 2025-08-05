package cfg

import (
	"fmt"
	"strings"

	"github.com/kballard/go-shellquote"
)

type Paramable interface {
	Params() []Param
	Param(string) Param
	SetParams(params ...Param) error
	HasParam(string) bool
	WasSet(string) bool
	Arg(string) any
	Args() map[string]any
	AllArgs() map[string]any // Returns both declared and pending args
	SetArg(string, any) error
	SetArgsFromList(args []string) error
}

type ParamHolder struct {
	params  map[string]Param
	pending map[string]*pendingArg
}

type pendingArg struct {
	Name    string
	Flag    string
	FromCLI bool
	Value   any
}

func newPendingArg(name string, value any) *pendingArg {
	return &pendingArg{Name: name, Value: value}
}

func (pa *pendingArg) SetFromCLI() {
	pa.FromCLI = true
}

func (pa *pendingArg) SetFlag(flag string) {
	pa.Flag = flag
}

func NewParamHolder() *ParamHolder {
	ph := &ParamHolder{
		params:  make(map[string]Param),
		pending: make(map[string]*pendingArg),
	}
	return ph
}

func (ph *ParamHolder) WasSet(name string) bool {
	param, ok := ph.getParam(name)
	if !ok {
		return false
	}
	return param.HasBeenSet()
}

func (ph *ParamHolder) Param(name string) Param {
	param, ok := ph.getParam(name)
	if !ok {
		return nil
	}
	return param
}

func (ph *ParamHolder) Params() []Param {
	params := []Param{}
	for _, param := range ph.params {
		params = append(params, param)
	}
	return params
}

func (ph *ParamHolder) HasParam(name string) bool {
	_, ok := ph.getParam(name)
	return ok
}

func (ph *ParamHolder) SetParams(params ...Param) error {
	for _, param := range params {
		if err := ph.SetParam(param); err != nil {
			return err
		}
	}
	return nil
}

func (ph *ParamHolder) SetParam(param Param) error {
	retrieved, ok := ph.getParam(param.Name())

	if ok && retrieved.Identifier() != param.Identifier() {
		return fmt.Errorf("param already exists with name %q", param.Name())
	}

	if err := param.isValidForShortcode(); err != nil {
		return fmt.Errorf("invalid shortcode for param %q: %w", param.Name(), err)
	}

	var err error

	pending, ok := ph.getPendingValue(param)
	if ok {
		param, err = param.SetValue(pending)
		if err != nil {
			return err
		}
		ph.deletePendingValue(param)
	}

	ph.params[param.Name()] = param
	return nil
}

func (ph *ParamHolder) getPendingValue(param Param) (any, bool) {
	pending, ok := ph.pending[param.Name()]
	if !ok {
		pending, ok = ph.pending[param.Flag()]
	}

	if ok {
		return pending.Value, true
	}

	return nil, false
}

func (ph *ParamHolder) deletePendingValue(param Param) {
	key := param.Name()
	_, ok := ph.pending[key]

	if !ok {
		key = param.Flag()
		_, ok = ph.pending[key]
	}

	if ok {
		delete(ph.pending, key)
	}
}

func (ph *ParamHolder) Arg(name string) any {
	param, ok := ph.getParam(name)
	if ok {
		return param.Value()
	}

	if pendingArg, exists := ph.pending[name]; exists {
		return pendingArg.Value
	}

	return nil
}

func (ph *ParamHolder) Args() map[string]any {
	args := map[string]any{}
	for name, param := range ph.params {
		if param.HasValue() {
			args[name] = param.Value()
		}
	}
	return args
}

func (ph *ParamHolder) AllArgs() map[string]any {
	args := map[string]any{}

	for name, param := range ph.params {
		if param.HasValue() {
			args[name] = param.Value()
		}
	}

	for name, pendingArg := range ph.pending {
		if _, exists := args[name]; !exists {
			args[name] = pendingArg.Value
		}
	}

	return args
}

func (ph *ParamHolder) SetArg(name string, value any) error {
	return ph.setArg(newPendingArg(name, value))
}

func (ph *ParamHolder) setArg(arg *pendingArg) error {
	param, ok := ph.getParamByFlag(arg.Name)
	if !ok {
		ph.pending[arg.Name] = arg
		return nil
	}

	param, err := param.SetValue(arg.Value)
	if err != nil {
		return err
	}

	ph.params[param.Name()] = param
	return nil
}

func (ph *ParamHolder) getParam(name string) (Param, bool) {
	param, ok := ph.params[name]
	return param, ok
}

func (ph *ParamHolder) getParamByFlag(flag string) (Param, bool) {
	param, ok := ph.getParam(flag)
	if ok {
		return param, true
	}

	for _, param := range ph.params {
		if param.Flag() == flag {
			return param, true
		}
	}

	return nil, false
}

func (ph *ParamHolder) SetArgsFromList(args []string) error {
	cli := shellquote.Join(args...)
	reparsed, err := shellquote.Split(cli)
	if err != nil {
		return fmt.Errorf("failed to split args: %w", err)
	}

	semiparsed := make(map[string][]string)
	currentFlag := ""
	for _, arg := range reparsed {
		if strings.HasPrefix(arg, "-") {
			for strings.HasPrefix(arg, "-") {
				arg = strings.TrimPrefix(arg, "-")
			}
			semiparsed[arg] = []string{}
			currentFlag = arg
		} else if _, ok := semiparsed[currentFlag]; !ok {
			return fmt.Errorf("encountered argument with no flag: %q", arg)
		} else {
			semiparsed[currentFlag] = append(semiparsed[currentFlag], arg)
		}
	}

	for flag, values := range semiparsed {
		value := strings.Join(values, ",")

		pendingArg := newPendingArg(flag, value)
		pendingArg.SetFlag(flag)
		pendingArg.SetFromCLI()

		if err := ph.setArg(pendingArg); err != nil {
			return err
		}
	}

	// flags := flag.NewFlagSet("args", flag.ContinueOnError)
	// stderr := &bytes.Buffer{}
	// flags.SetOutput(stderr)

	// for _, param := range ph.params {
	// 	flagName := param.Shortcode()
	// 	flags.String(flagName, "", param.Description())
	// }
	// err := flags.Parse(args)
	// if err != nil {
	// 	formatted := strings.Replace(stderr.String(), "Usage of args", "Chain Parameters", 1)
	// 	formatted = strings.Replace(formatted, "flag provided but not defined", "unknown/unsupported parameter provided", 1)
	// 	return fmt.Errorf("%s", formatted)
	// }

	// for _, param := range ph.params {
	// 	setByUser := slices.Contains(args, "-"+param.Shortcode())
	// 	if err := ph.setArgFromFlag(flags, param, setByUser); err != nil {
	// 		return err
	// 	}
	// }

	return nil
}

// func (ph *ParamHolder) setArgFromFlag(flags *flag.FlagSet, param Param, setByUser bool) error {
// 	paramFlag := flags.Lookup(param.Shortcode())
// 	if paramFlag == nil {
// 		slog.Warn("did not find flag for param", "param", param.Name())
// 		return nil
// 	}

// 	unconverted := paramFlag.Value
// 	if !setByUser && unconverted.String() == "" {
// 		return nil // flags.Lookup() reports an empty string for missing flags; ignore these
// 	}

// 	arg, err := param.convertFromCLIString(unconverted.String())
// 	if err != nil {
// 		return fmt.Errorf("failed to convert value %q to type %q: %w", unconverted.String(), param.Type(), err)
// 	}

// 	newParam, err := ph.params[param.Name()].SetValue(arg)
// 	if err != nil {
// 		return err
// 	}

// 	ph.params[param.Name()] = newParam
// 	return nil
// }

func (ph *ParamHolder) Validate() error {
	for _, param := range ph.params {
		if !param.isSettableTo(param.Value()) {
			return fmt.Errorf("parameter %q expects type %q, but argument value is type %q", param.Name(), param.Type(), fmt.Sprintf("%T", param.Value()))
		}

		if err := param.isValidForRegex(param.Value()); err != nil {
			return fmt.Errorf("error validating regex: %w", err)
		}

		if param.Required() && !param.HasBeenSet() && !param.HasDefault() {
			return fmt.Errorf("parameter %q is required", param.Name())
		}
	}
	return nil
}
