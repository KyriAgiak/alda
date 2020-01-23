package emitter

import (
	"fmt"
	"math"
	"time"

	"alda.io/client/model"
	"github.com/daveyarwood/go-osc/osc"
)

// OSCEmitter sends OSC messages to a player process.
type OSCEmitter struct {
	Port int
}

func systemPlayMsg() *osc.Message {
	return osc.NewMessage("/system/play")
}

func midiPatchMsg(track int32, offset int32, patch int32) *osc.Message {
	msg := osc.NewMessage(fmt.Sprintf("/track/%d/midi/patch", track))
	msg.Append(offset)
	msg.Append(patch)
	return msg
}

func midiPercussionMsg(track int32, offset int32) *osc.Message {
	msg := osc.NewMessage(fmt.Sprintf("/track/%d/midi/percussion", track))
	msg.Append(offset)
	return msg
}

func midiNoteMsg(
	track int32, offset int32, note int32, duration int32, audibleDuration int32,
	velocity int32,
) *osc.Message {
	msg := osc.NewMessage(fmt.Sprintf("/track/%d/midi/note", track))
	msg.Append(offset)
	msg.Append(note)
	msg.Append(duration)
	msg.Append(audibleDuration)
	msg.Append(velocity)
	return msg
}

// EmitScore implements Emitter.EmitScore by sending OSC messages to instruct a
// player process how to perform the score.
func (oe OSCEmitter) EmitScore(score *model.Score) error {
	client := osc.NewClient("localhost", int(oe.Port))
	client.SetNetworkProtocol(osc.TCP)
	bundle := osc.NewBundle(time.Now())

	tracks := score.Tracks()

	for part, trackNumber := range tracks {
		// We currently only have MIDI instruments. This might change in the future,
		// which is why Instrument is an interface instead of a plain struct. For
		// now, we're operating under the assumption that all instruments are MIDI
		// instruments.
		stockInstrument := part.StockInstrument.(model.MidiInstrument)

		patchNumber := stockInstrument.PatchNumber
		bundle.Append(midiPatchMsg(trackNumber, 0, patchNumber))

		if stockInstrument.IsPercussion {
			bundle.Append(midiPercussionMsg(trackNumber, 0))
		}
	}

	for _, event := range score.Events {
		switch event.(type) {
		case model.NoteEvent:
			noteEvent := event.(model.NoteEvent)
			bundle.Append(midiNoteMsg(
				tracks[noteEvent.Part],
				int32(math.Round(float64(noteEvent.Offset))),
				noteEvent.MidiNote,
				int32(math.Round(float64(noteEvent.Duration))),
				int32(math.Round(float64(noteEvent.AudibleDuration))),
				int32(math.Round(float64(noteEvent.Volume*127))),
				// TODO: handle track volume, panning
				// I'm thinking these should be separate types of OSC message, like
				// /track/1/midi/volume and /track/1/midi/panning. In the MIDI spec,
				// they are sent separately from notes as control change messages.
			))
		default:
			return fmt.Errorf("unsupported event: %#v", event)
		}
	}

	bundle.Append(systemPlayMsg())

	client.Send(bundle)

	return nil
}