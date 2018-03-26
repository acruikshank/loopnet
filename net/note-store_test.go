package loopnet

import (
  "testing"
  "fmt"
  "reflect"
  p2p "github.com/acruikshank/loopnet/pb"
)

func TestNoteStore(t *testing.T) {
  selfNote := createNote("self", 0, 63, false)
  node1Note := createNote("node1", 43, 65, false)

  t.Run("onNote", func(t *testing.T) {
    noteStore := NewNoteStore(selfNote)

    t.Run("NoteStore stores notes from new nodes", func(t *testing.T) {
      if noteStore.ActiveNotes() != 1 {
        t.Error("noteStore is not initialized with self")
      }

      noteStore.OnNote(*node1Note)

      if noteStore.ActiveNotes() != 2 {
        t.Error("noteStore did not add other note")
      }
    })

    t.Run("Update revision for new node", func(t *testing.T) {
      noteStore.OnNote(*node1Note)
      noteStore.OnNote(*createNote("node1",44, 68, false))

      note, ok := noteStore.LastRevision("node1")
      if !ok {
        t.Error("did not store note")
      }

      if note.Note != 68 || note.Revision != 44 {
        t.Error("did not store latest revision")
      }
    })

    // noteStore = NewNoteStore(selfNote)
    t.Run("Ignores old revision", func(t *testing.T) {
      noteStore.OnNote(*node1Note)
      noteStore.OnNote(*createNote("node1",42, 61, false))

      note, ok := noteStore.LastRevision("node1")
      if !ok {
        t.Error("did not store note")
      }

      if note.Note == 61 || note.Revision == 42 {
        t.Error("stored older revision when it should have been ignored")
      }
    })
  })

  t.Run("RandomNotes", func(t *testing.T) {
    noteStore := NewNoteStore(selfNote)

    t.Run("eventually returns all the notes", func(t *testing.T) {
      allNotes := createNotes(10)
      seenIds := make(map[string]bool)
      for _, note := range allNotes {
        noteStore.OnNote(note)
        seenIds[note.NodeId] = true
      }

      // this should produce a false positive once every billion or so runs
      for i := 0; i < 200; i++ {
        randNotes := noteStore.RandomNotes(2, false)

        if len(randNotes) != 2 {
          t.Error("wrong number of notes returned.")
        }
        for _, note := range(randNotes) {
          delete(seenIds, note.NodeId)
        }
      }

      if len(seenIds) > 0 {
        t.Error("failed to eventually return all active notes")
      }
    })


    t.Run("never returns self when self is excluded", func(t *testing.T) {
      allNotes := createNotes(10)
      seenIds := make(map[string]bool)
      for _, note := range allNotes {
        noteStore.OnNote(note)
        seenIds[note.NodeId] = true
      }

      // this should produce a false positive once every billion or so runs
      for i := 0; i < 200; i++ {
        randNotes := noteStore.RandomNotes(2, true)

        for _, note := range(randNotes) {
          if note.NodeId == "self" {
            t.Error("failed to exclude self")
            t.FailNow()
          }
        }
      }
    })
  })

  t.Run("ActiveNoteNumbers", func(t *testing.T) {
    noteStore := NewNoteStore(selfNote)

    t.Run("NoteStore numbers for each unmuted note, sorted", func(t *testing.T) {
      noteStore.OnNote(*createNote("n1", 1, 32, false))
      noteStore.OnNote(*createNote("n2", 1, 72, false))
      noteStore.OnNote(*createNote("n3", 1, 12, false))
      noteStore.OnNote(*createNote("n4", 1, 31, true))
      noteStore.OnNote(*createNote("n5", 1, 64, false))
      noteStore.OnNote(*createNote("n6", 1, 18, true))

      noteNumbers := noteStore.ActiveNoteNumbers()
      expectation := []int{12, 32, 63, 64, 72}
      if !reflect.DeepEqual(noteNumbers, expectation) {
        t.Errorf("Expected %v notes, got %v", expectation, noteNumbers)
      }
    })
  })

  t.Run("ClearDeadNotes", func(t *testing.T) {
    noteStore := NewNoteStore(selfNote)

    t.Run("removes nodes that have fallen behind in revisions", func(t *testing.T) {
      noteStore.OnNote(*createNote("n1", 1, 32, false))
      noteStore.OnNote(*createNote("n2", 1, 33, false))
      noteStore.OnNote(*createNote("n3", 1, 34, false))
      noteStore.OnNote(*createNote("self", 1, 64, true)) // mute self

      noteNumbers := noteStore.ActiveNoteNumbers()
      expectation := []int{32,33,34}
      if !reflect.DeepEqual(noteNumbers, expectation) {
        t.Errorf("Expected initial %v notes, got %v", expectation, noteNumbers)
      }

      for i := uint32(2); i < 50; i++ {
        noteStore.OnNote(*createNote("n2", i, 33, false))
        noteStore.OnNote(*createNote("n3", i, 34, false))
      }

      noteStore.ClearDeadNotes()

      noteNumbers = noteStore.ActiveNoteNumbers()
      expectation = []int{33,34}
      if !reflect.DeepEqual(noteNumbers, expectation) {
        t.Errorf("Expected to have dropped one note (expected %v notes, got %v)", expectation, noteNumbers)
      }
    })
  })
}

func createNote(node string, revision uint32, note uint32, muted bool) *p2p.NoteData {
  return &p2p.NoteData {
    Address: "/ip4/127.0.0.1/tcp/1000",
    NodeId: node,
    Revision: revision,
    Note: note,
    Mute: muted,
  }
}

func createNotes(count int) []p2p.NoteData {
  notes := make([]p2p.NoteData, count)
  for i := 0; i < count; i++ {
    notes[i] = *createNote(fmt.Sprintf("node%d", i), uint32(i), uint32(20+i), false)
  }
  return notes
}
