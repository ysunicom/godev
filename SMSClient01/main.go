package main

import (
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/linxGnu/gosmpp"
	"github.com/linxGnu/gosmpp/data"
	"github.com/linxGnu/gosmpp/pdu"
)

func main() {
	var wg sync.WaitGroup
	tempmessage := flag.String("message", "Hello world", "消息体")
	tempaddr := flag.String("address", ":2002", "SMSC地址或代理地址")
	flag.Parse()
	message := *tempmessage
	address := *tempaddr
	wg.Add(1)
	go sendingAndReceiveSMS(&wg, message, address)

	wg.Wait()
}

func sendingAndReceiveSMS(wg *sync.WaitGroup, message, address string) {
	defer wg.Done()
	auth := gosmpp.Auth{
		SMSC:       address,
		SystemID:   "YJDX",
		Password:   "YJdx!2",
		SystemType: "",
	}
	var ecmClass_long, ecmClass_short byte
	ecmClass_long = 64 //长短信要设置成这个值
	ecmClass_short = 0 //一般短信设置成这个值
	trans, err := gosmpp.NewTransceiverSession(gosmpp.NonTLSDialer, auth, gosmpp.TransceiveSettings{
		EnquireLink: 5 * time.Second,

		OnSubmitError: func(p pdu.PDU, err error) {
			log.Fatal(err)
		},

		OnReceivingError: func(err error) {
			fmt.Println("receive error:", err)
		},

		OnRebindingError: func(err error) {
			fmt.Println("binding error:", err)
		},

		OnPDU: handlePDU(),

		OnClosed: func(state gosmpp.State) {
			fmt.Println("state:", state)
		},
	}, 5*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = trans.Close()
	}()
	if len(message) > 140 {
		//log.Println("submit long shortmessage")
		shortMessageSlice, _ := pdu.NewLongMessageWithEncoding(message, data.UCS2)
		// sending SMS(s)
		for _, v := range shortMessageSlice {
			if err = trans.Transceiver().Submit(newSubmitSM("18607181308", ecmClass_long, v)); err != nil {
				fmt.Println(err)
			}
			time.Sleep(time.Second)
		}
	} else {
		//log.Println("submit shortmessage")
		shortMessage, _ := pdu.NewShortMessageWithEncoding(message, data.UCS2)
		if err = trans.Transceiver().Submit(newSubmitSM("18607181308", ecmClass_short, &shortMessage)); err != nil {
			fmt.Println(err)
		}
	}

}

func handlePDU() func(pdu.PDU, bool) {
	return func(p pdu.PDU, responded bool) {
		switch pd := p.(type) {
		case *pdu.SubmitSMResp:
			fmt.Printf("SubmitSMResp:%+v\n", pd)

		case *pdu.GenerickNack:
			fmt.Println("GenericNack Received")

		case *pdu.EnquireLinkResp:
			fmt.Println("EnquireLinkResp Received")

		case *pdu.DataSM:
			fmt.Printf("DataSM:%+v\n", pd)

		case *pdu.DeliverSM:
			fmt.Printf("DeliverSM:%+v\n", pd)
			fmt.Println(pd.Message.GetMessage())
		}
	}
}

func newSubmitSM(destNum string, ecmClass byte, message *pdu.ShortMessage) *pdu.SubmitSM {
	// build up submitSM
	srcAddr := pdu.NewAddress()
	srcAddr.SetTon(2)
	srcAddr.SetNpi(1)
	_ = srcAddr.SetAddress("10010")

	destAddr := pdu.NewAddress()
	destAddr.SetTon(1)
	destAddr.SetNpi(1)
	_ = destAddr.SetAddress(destNum)

	submitSM := pdu.NewSubmitSM().(*pdu.SubmitSM)
	submitSM.SourceAddr = srcAddr
	submitSM.DestAddr = destAddr
	//_ = submitSM.Message.SetLongMessageWithEnc(message, data.UCS2)
	submitSM.Message = *message
	submitSM.ProtocolID = 0
	submitSM.RegisteredDelivery = 1
	submitSM.ReplaceIfPresentFlag = 0
	submitSM.EsmClass = ecmClass

	return submitSM
}
