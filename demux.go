package binproto

type urReturn struct {
	ut   UnitType
	data interface{}
}

type Demux struct {
	ur            UnitReader
	events, other chan urReturn
	err           error
}

func NewDemux(ur UnitReader) (d *Demux) {
	d = &Demux{
		ur:     ur,
		events: make(chan urReturn),
		other:  make(chan urReturn),
		err:    nil}
	go d.demux()
	return
}

func (d *Demux) demux() {
	inEvent := false
	nesting := 0

	for {
		ut, data, err := d.ur.ReadUnit()
		if err != nil {
			d.err = err
			close(d.events)
			close(d.other)
			return
		}

		if inEvent {
			switch ut {
			case UTList, UTIdKVMap, UTTextKVMap:
				nesting++
			case UTTerm:
				nesting--
			}

			d.events <- urReturn{ut, data}

			if nesting <= 0 {
				inEvent = false
			}
		} else if ut == UTEvent {
			d.events <- urReturn{ut, data}
			inEvent = true
			nesting = 0
		} else {
			d.other <- urReturn{ut, data}
		}
	}
}

type PartUnitReader struct {
	ch chan urReturn
	d  *Demux
}

func (d *Demux) Events() *PartUnitReader { return &PartUnitReader{d.events, d} }
func (d *Demux) Other() *PartUnitReader  { return &PartUnitReader{d.other, d} }

func (pur *PartUnitReader) ReadUnit() (UnitType, interface{}, error) {
	urr, ok := <-pur.ch
	if !ok {
		return 0, nil, pur.d.err
	}

	return urr.ut, urr.data, nil
}
