package loopnet

import (
	"crypto/rand"
	p2p "github.com/acruikshank/loopnet/pb"
	"math/big"
	"sort"
	"sync"
)

const deadNoteRevisions = 20

type Note struct {
	revision uint32
	*p2p.NoteData
}

type NoteStore struct {
	selfId            string
	referenceRevision uint32
	notes             map[string]Note
	noteMux           *sync.RWMutex
}

// NewNoteStore creates a new store with the initial revision of the local node's note.
func NewNoteStore(self *p2p.NoteData) *NoteStore {
	n := &NoteStore{
		selfId:            self.NodeId,
		referenceRevision: 0,
		notes:             make(map[string]Note),
		noteMux:           &sync.RWMutex{},
	}
	n.notes[self.NodeId] = Note{
		revision: 0,
		NoteData: self,
	}
	return n
}

// OnNote takes a note from a node and adds it to the store if it represents a new
// note or if its revision is higher than the revision currently stored.
func (ns *NoteStore) OnNote(note p2p.NoteData) bool {
	ns.noteMux.Lock()
	defer ns.noteMux.Unlock()

	existingNote, found := ns.notes[note.NodeId]
	if found {
		// ignore stale information
		if existingNote.Revision >= note.Revision {
			return false
		}

		// start a new referenceRevision round if this node is up-to-date
		if existingNote.revision >= ns.referenceRevision {
			ns.referenceRevision++
		}
	}

	// add or update node
	ns.notes[note.NodeId] = Note{
		revision: ns.referenceRevision,
		NoteData: &note,
	}

	return !found
}

// returns a slice of notes chosen randomly from active notes.
// The count parameter specifies the number of random nodes. If it exceeds
// the number available, only the number available will be returned.
// If excludeSelf is true, the note for this node will never be included.
func (ns *NoteStore) RandomNotes(count int, excludeSelf bool) []*p2p.NoteData {
	ns.noteMux.RLock()
	defer ns.noteMux.RUnlock()

	keys := make([]string, 0)
	for nodeId := range ns.notes {
		if !excludeSelf || nodeId != ns.selfId {
			keys = append(keys, nodeId)
		}
	}

	if count > len(keys) {
		count = len(keys)
	}

	out := make([]*p2p.NoteData, 0)
	for i := 0; i < count; i++ {
		index := randomInt(len(keys))
		out = append(out, ns.notes[keys[index]].NoteData)
		keys = append(keys[:index], keys[index+1:]...)
	}

	return out
}

// ClearDeadNotes removes any note that appears to be dead.
// Dead notes are identified as any note that has failed to
// update within the time it has taken us to see some number
// of updates (e.g. 20) by the most frequently updated note.
func (ns *NoteStore) ClearDeadNotes() {
	ns.noteMux.Lock()
	defer ns.noteMux.Unlock()

	deadNotes := make(map[string]bool)
	for nodeId, note := range ns.notes {
		if ns.referenceRevision-note.revision > deadNoteRevisions {
			deadNotes[nodeId] = true
		}
	}

	if len(deadNotes) > 0 {
		for nodeId := range deadNotes {
			delete(ns.notes, nodeId)
		}
	}
}

// ActiveNoteNumbers returns a sorted list of all the midi
// note number of all currently stored notes that are not
// muted.
func (ns *NoteStore) ActiveNoteNumbers() []int {
	ns.noteMux.RLock()
	defer ns.noteMux.RUnlock()

	noteNumbers := make([]int, 0)
	for _, note := range ns.notes {
		if !note.Mute {
			noteNumbers = append(noteNumbers, int(note.Note))
		}
	}

	sort.Ints(noteNumbers)

	return noteNumbers
}

// ActiveNotes returns the number of currently stored notes.
func (ns *NoteStore) ActiveNotes() int {
	ns.noteMux.RLock()
	defer ns.noteMux.RUnlock()

	return len(ns.notes)
}

// LastRevision takes a node id and returns whether the note
// is currently being stored and its note message if so.
func (ns *NoteStore) LastRevision(nodeId string) (p2p.NoteData, bool) {
	ns.noteMux.RLock()
	defer ns.noteMux.RUnlock()

	note, ok := ns.notes[nodeId]
	if !ok {
		return p2p.NoteData{}, ok
	}
	return *note.NoteData, ok
}

func randomInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		panic("Error fetching random number")
	}
	return int(n.Int64())
}
