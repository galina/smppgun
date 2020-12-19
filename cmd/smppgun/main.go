package main

import (
	"time"

	"github.com/spf13/afero"
	"github.com/yandex/pandora/cli"
	"github.com/yandex/pandora/core"
	"github.com/yandex/pandora/core/aggregator/netsample"
	"github.com/yandex/pandora/core/import"
	"github.com/yandex/pandora/core/provider"
	"github.com/yandex/pandora/core/register"

	"github.com/fiorix/go-smpp/smpp"
	"github.com/fiorix/go-smpp/smpp/pdu"
	"github.com/fiorix/go-smpp/smpp/pdu/pdufield"
)

type Ammo struct {
	Tag  string
	Text string
	Src  string
	Dst  string
	Enc  string

	parts []smpp.ShortMessage
}

type SmppGunConfig struct {
	Target          string `validate:"required"`
	SystemId        string
	Password        string
	DeliveryReceipt pdufield.DeliverySetting
	Esme            EsmeConfig
}

type EsmeConfig struct {
	EnquireLink        time.Duration
	EnquireLinkTimeout time.Duration
	RespTimeout        time.Duration
	BindInterval       time.Duration
	WindowSize         uint
}

type Gun struct {
	conf   SmppGunConfig
	client *smpp.Transceiver
	aggr   core.Aggregator
	core.GunDeps
}

func NewGun(conf SmppGunConfig) *Gun {
	return &Gun{conf: conf}
}

func (g *Gun) Bind(aggr core.Aggregator, deps core.GunDeps) error {
	conn, err := bindTransceiver(g.conf, aggr)
	if err != nil {
		return err
	}

	g.client = conn
	g.aggr = aggr
	g.GunDeps = deps
	return nil
}

func (g *Gun) Shoot(ammo core.Ammo) {
	smppAmmo := ammo.(*Ammo)
	g.shoot(smppAmmo)
}

func bindTransceiver(conf SmppGunConfig, aggr core.Aggregator) (*smpp.Transceiver, error) {
	f := func(p pdu.Body) {
		switch p.Header().ID {
		case pdu.DeliverSMID:
			sample := netsample.Acquire("dlr")
			aggr.Report(sample)
		}
	}

	trx := &smpp.Transceiver{
		Addr:               conf.Target,
		User:               conf.SystemId,
		Passwd:             conf.Password,
		Handler:            f,
		RespTimeout:        conf.Esme.RespTimeout,
		BindInterval:       conf.Esme.BindInterval,
		WindowSize:         conf.Esme.WindowSize,
		EnquireLink:        conf.Esme.EnquireLink,
		EnquireLinkTimeout: conf.Esme.EnquireLinkTimeout,
	}

	return trx, bind(trx)
}

func bind(client smpp.ClientConn) error {
	conn := client.Bind()
	var status smpp.ConnStatus
	if status = <-conn; status.Error() != nil {
		return status.Error()
	}
	return nil
}

func DefaultSmppGunConfig() SmppGunConfig {
	return SmppGunConfig{
		SystemId:        "test",
		Password:        "test",
		DeliveryReceipt: pdufield.FinalDeliveryReceipt,
	}
}

func (g *Gun) shoot(ammo *Ammo) {
	for _, part := range ammo.parts {
		part.Register = g.conf.DeliveryReceipt

		sample := netsample.Acquire(ammo.Tag)
		g.submit(sample, &part)
	}
}

func (g *Gun) submit(sample *netsample.Sample, sm *smpp.ShortMessage) {
	defer func() {
		g.aggr.Report(sample)
	}()

	if sm, err := g.client.Submit(sm); err != nil {
		handleError(sample, err)
	} else {
		resp := sm.Resp()
		sample.SetProtoCode(int(resp.Header().Status))
	}
}

func handleError(sample *netsample.Sample, err error) {
	switch err {
	case smpp.ErrNotConnected:
		panic("not connected")
	default:
		sample.SetErr(err)
	}
}

func main() {
	fs := afero.NewOsFs()
	coreimport.Import(fs)

	wrapDecoder := func(deps core.ProviderDeps, decoder provider.AmmoDecoder) provider.AmmoDecoder {
		return provider.AmmoDecoderFunc(func(ammo core.Ammo) error {
			err := decoder.Decode(ammo)
			if err != nil {
				return err
			}

			smppAmmo := ammo.(*Ammo)

			smppAmmo.parts = SplitMessageText(&smpp.ShortMessage{
				Src: smppAmmo.Src,
				Dst: smppAmmo.Dst,
			}, smppAmmo.Text, smppAmmo.Enc)

			return nil
		})
	}

	newAmmo := func() core.Ammo { return &Ammo{} }

	register.Provider("smpp_provider", func(conf provider.JSONProviderConfig) core.Provider {
		return provider.NewCustomJSONProvider(wrapDecoder, newAmmo, conf)
	}, provider.DefaultJSONProviderConfig)

	register.Gun("smpp", NewGun, DefaultSmppGunConfig)

	cli.Run()
}
