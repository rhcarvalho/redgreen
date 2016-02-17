package sound

import (
	"bytes"
	"fmt"
)

func Example() {
	sounds := []struct {
		name string
		buf  bytes.Buffer
	}{
		{"Explosive counter", ExplosiveCounter},
		{"Super Nintendo", SuperNintendo},
		// {"Phone ring", PhoneRing},
		{"Fluid", Fluid},
	}
	Say(fmt.Sprintf("Hello, I am going to play %d example sounds!", len(sounds)))
	for i, snd := range sounds {
		Say(fmt.Sprintf("%d, %s:", i+1, snd.name))
		Play(&snd.buf)
	}
	// Output:
}
