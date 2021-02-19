package cardscanner

import (
	//"github.com/warthog618/gpiod"
	//"github.com/ecc1/spi"

	"errors"
	"log"
	"strconv"

	"github.com/ecc1/spi"
)

//Card -
type Card struct {
	spiDevice     *spi.Device
	speed         int
	spiDeviceAddr string
}

//CardReaderIO - Interface for Scanner
type CardReaderIO interface {
	Capture() error
	Release()
	VerifyPassword() bool
	Scan() (string, error)
	Flash(data []byte) error
	RequestMode(int) (tagType int, err error)
	ReadWithAnticoll() (uid string, err error)
}

//NewCardScanner - Params - spiDeviceAddr > /dev/spidev0.0, speed > 100000
func NewCardScanner(spiDeviceAddr string, speed int) CardReaderIO {
	return &Card{spiDeviceAddr: spiDeviceAddr, speed: speed}

}

//Capture -
func (c *Card) Capture() error {
	err := error(nil)
	c.spiDevice, err = spi.Open(c.spiDeviceAddr, c.speed, 0)
	if err != nil {
		log.Printf(err.Error())
		return err
	}
	return nil
}

//Release -
func (c *Card) Release() {
	var outBuf []byte
	outBuf = append(outBuf, 0)
	c.spiDevice.Write(outBuf)
}

//VerifyPassword -
func (c *Card) VerifyPassword() bool {
	return true
}

//Scan -
func (c *Card) Scan() (string, error) {
	return "asasas", nil
}

//Flash -
func (c *Card) Flash(data []byte) error {
	return nil
}

func (c *Card) writeToDevice(addr int, val int) (responseBytes []byte, err error) {
	var outBuf []byte

	responseBytes = nil

	bAddr := []byte(strconv.Itoa(((addr << 1) & 0x7E)))
	bVal := []byte(strconv.Itoa(val))
	outBuf = append(outBuf, bAddr[:]...)
	outBuf = append(outBuf, bVal[:]...)
	err = c.spiDevice.Transfer(outBuf)
	if err == nil {
		responseBytes = outBuf
	}
	return responseBytes, err
}

func (c *Card) setBitMask(reg int, mask int) {
	tmp, _ := c.readFromDevice(reg)
	c.writeToDevice(reg, int(tmp)|mask)
}

func (c *Card) clearBitMask(reg int, mask int) {
	tmp, _ := c.readFromDevice(reg)
	c.writeToDevice(reg, int(tmp)&^mask)
}

func (c *Card) readFromDevice(addr int) (byte, error) {
	responseBytes, err := c.writeToDevice(addr, 0)
	return responseBytes[1], err
}

func (c *Card) writeCommandToCard(command int, sendData []byte) (responseData []byte, responseLen int, err error) {

	irqEn := 0x00
	waitIRq := 0x00
	status := MI_ERR

	responseLen = 0
	responseData = make([]byte, 0)

	// lastBits := None

	if command == PCD_AUTHENT {
		irqEn = 0x12
		waitIRq = 0x10
	} else if command == PCD_TRANSCEIVE {
		irqEn = 0x77
		waitIRq = 0x30
	}

	c.writeToDevice(CommIEnReg, irqEn|0x80)
	c.clearBitMask(CommIrqReg, 0x80)
	c.setBitMask(FIFOLevelReg, 0x80)

	c.writeToDevice(CommandReg, PCD_IDLE)
	i := 0
	for i < len(sendData) {
		c.writeToDevice(FIFODataReg, int(sendData[i]))
		i = i + 1
	}

	c.writeToDevice(CommandReg, command)

	if command == PCD_TRANSCEIVE {
		c.setBitMask(BitFramingReg, 0x80)
	}

	i = 2000
	n := byte(0)
	for {
		n, _ = c.readFromDevice(CommandReg)
		i = i - 1
		//(i != 0) && ^(int(n) & 0x01) && ^(int(n) & waitIRq)
		if (i != 0) && (^(int(n) & 0x01)) != 0 && (^(int(n) & waitIRq)) != 0 {
			break
		}
	}

	c.clearBitMask(BitFramingReg, 0x80)

	if i != 0 {
		b, _ := c.readFromDevice(ErrorReg)
		if (b & 0x1B) == 0x00 {
			status = MI_OK

			if int(n) != 0&irqEn&0x01 {
				status = MI_NOTAGERR
			}

			if command == PCD_TRANSCEIVE {
				n, _ = c.readFromDevice(FIFOLevelReg)
				lastBits, _ := c.readFromDevice(ControlReg)
				lastBitsInteger := int(lastBits) & 0x07
				if lastBits != 0 {
					responseLen = (int(n)-1)*8 + lastBitsInteger
				} else {
					responseLen = int(n) * 8
				}
			}

			if n == 0 {
				n = 1
			}
			if n > MAX_LEN {
				n = MAX_LEN
			}

			i = 0
			for i < int(n) {
				resp, _ := c.readFromDevice(FIFODataReg)
				responseData = append(responseData, resp)
				i = i + 1
			}
		}
	} else {
		status = MI_ERR
	}

	if status != MI_OK {
		err = errors.New("Some error")
		responseData = nil
		responseLen = -1
	}

	return responseData, responseLen, err
}

//RequestMode -
func (c *Card) RequestMode(reqMode int) (int, error) {

	err := error(nil)
	outBuf := make([]byte, 0)

	c.writeToDevice(BitFramingReg, 0x07)
	outBuf = append(outBuf, byte(reqMode))

	responseBytes, responseLen, err := c.writeCommandToCard(PCD_TRANSCEIVE, outBuf)

	if (err != nil) || (responseLen != 0x10) {
		return 0, err
	}
	log.Println("Response ->", responseBytes)

	return responseLen, err
}

//ReadWithAnticoll -
func (c *Card) ReadWithAnticoll() (uid string, err error) {

	return "nil", nil
}