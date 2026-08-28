package ui

type ReconcileKind uint8

const (
	ReconcileKeep ReconcileKind = iota
	ReconcileInsert
	ReconcileMove
	ReconcileRemove
)

type ReconcileOperation struct {
	Kind ReconcileKind
	ID   string
	From int
	To   int
}

func ReconcileRows(current, next []string) []ReconcileOperation {
	wanted := make(map[string]bool, len(next))
	for _, id := range next {
		wanted[id] = true
	}
	working := append([]string(nil), current...)
	operations := make([]ReconcileOperation, 0, len(current)+len(next))
	for index := len(working) - 1; index >= 0; index-- {
		if wanted[working[index]] {
			continue
		}
		operations = append(operations, ReconcileOperation{Kind: ReconcileRemove, ID: working[index], From: index, To: -1})
		working = append(working[:index], working[index+1:]...)
	}
	for target, id := range next {
		if target < len(working) && working[target] == id {
			operations = append(operations, ReconcileOperation{Kind: ReconcileKeep, ID: id, From: target, To: target})
			continue
		}
		from := indexOfID(working, id, target)
		if from >= 0 {
			operations = append(operations, ReconcileOperation{Kind: ReconcileMove, ID: id, From: from, To: target})
			working = append(working[:from], working[from+1:]...)
			working = insertID(working, target, id)
			continue
		}
		operations = append(operations, ReconcileOperation{Kind: ReconcileInsert, ID: id, From: -1, To: target})
		working = insertID(working, target, id)
	}
	return operations
}

func indexOfID(ids []string, id string, start int) int {
	for index := start; index < len(ids); index++ {
		if ids[index] == id {
			return index
		}
	}
	return -1
}

func insertID(ids []string, index int, id string) []string {
	ids = append(ids, "")
	copy(ids[index+1:], ids[index:])
	ids[index] = id
	return ids
}
