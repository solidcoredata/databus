package typespec

type Slot struct {
	Name     string
	SrcName  string
	DestName string
}
type Field struct {
	Name     string
	Required bool
	Repeat   bool
	Position int
	Resolve  []interface{} // I don't yet know what type.
}
type Type struct {
	Name   string
	Slot   []*Slot
	Fields []*Field
}

type Comments struct {
	Before    string
	EndOfLine string
}

type SlotValue struct {
	SlotName string
	Slot     *Slot
}

type FieldValue struct {
	FieldName string
	Field     *Field

	RawAtom string
	Atom    interface{} // Parsed vvalue.

	FieldValues []*FieldValue
}

type Instance struct {
	Comments Comments

	TypeName string
	Type     *Type

	SlotValues  []*SlotValue
	FieldValues []*FieldValue
}

var root = []*Instance{
	{
		TypeName: "SearchListDetail",
	},
}
