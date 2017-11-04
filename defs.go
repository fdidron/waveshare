// functions related to communication, commands protocols for the EPD
package waveshare

import (
	"flag"
	"image"
	"image/color"
	"log"
	"strconv"
	"time"

	"github.com/golang/glog"

	"github.com/kidoman/embd"
)

// # Display resolution
var EPD_WIDTH byte = 200
var EPD_HEIGHT byte = 200

// # EPD1IN54 commands
var DRIVER_OUTPUT_CONTROL byte = 0x01
var BOOSTER_SOFT_START_CONTROL byte = 0x0C
var GATE_SCAN_START_POSITION byte = 0x0F
var DEEP_SLEEP_MODE byte = 0x10
var DATA_ENTRY_MODE_SETTING byte = 0x11
var SW_RESET byte = 0x12
var TEMPERATURE_SENSOR_CONTROL byte = 0x1A
var MASTER_ACTIVATION byte = 0x20
var DISPLAY_UPDATE_CONTROL_1 byte = 0x21
var DISPLAY_UPDATE_CONTROL_2 byte = 0x22
var WRITE_RAM byte = 0x24
var WRITE_VCOM_REGISTER byte = 0x2C
var WRITE_LUT_REGISTER byte = 0x32
var SET_DUMMY_LINE_PERIOD byte = 0x3A
var SET_GATE_TIME byte = 0x3B
var BORDER_WAVEFORM_CONTROL byte = 0x3C
var SET_RAM_X_ADDRESS_START_END_POSITION byte = 0x44
var SET_RAM_Y_ADDRESS_START_END_POSITION byte = 0x45
var SET_RAM_X_ADDRESS_COUNTER byte = 0x4E
var SET_RAM_Y_ADDRESS_COUNTER byte = 0x4F
var TERMINATE_FRAME_READ_WRITE byte = 0xFF

func init() {
	flag.Parse()
}

type EPD struct {
	lutFull bool
	// Sequence for updating
	lutFullUpdate    []byte
	lutPartialUpdate []byte
}

func (e *EPD) SetDefaults() {
	e.lutFullUpdate = []byte{
		0x02, 0x02, 0x01, 0x11, 0x12, 0x12, 0x22, 0x22,
		0x66, 0x69, 0x69, 0x59, 0x58, 0x99, 0x99, 0x88,
		0x00, 0x00, 0x00, 0x00, 0xF8, 0xB4, 0x13, 0x51,
		0x35, 0x51, 0x51, 0x19, 0x01, 0x00}

	e.lutPartialUpdate = []byte{
		0x10, 0x18, 0x18, 0x08, 0x18, 0x18, 0x08, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x13, 0x14, 0x44, 0x12,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

	e.lutFull = true
}

func (e *EPD) SendCommand(command byte) {

	embd.DigitalWrite(DC_PIN, embd.Low)
	spibus.Write([]byte{command})

}

func (e *EPD) SendData(data ...byte) {
	embd.DigitalWrite(DC_PIN, embd.High)
	spibus.Write(data)

}

func (e *EPD) CallFunction(command byte, data ...byte) {
	e.SendCommand(command)
	e.SendData(data...)
}

func (e *EPD) Init(full bool) {

	var dataseq []byte
	e.lutFull = full
	// self.lut = lut
	// self.reset()
	e.reset()
	// self.send_command(DRIVER_OUTPUT_CONTROL)
	// self.send_data((EPD_HEIGHT - 1) & 0xFF)
	// self.send_data(((EPD_HEIGHT - 1) >> 8) & 0xFF)
	// self.send_data(0x00)                     # GD = 0 SM = 0 TB = 0
	dataseq = []byte{(EPD_HEIGHT - 1) & 0xFF, ((EPD_HEIGHT - 1) >> 8) & 0xFF, 0x00}
	e.CallFunction(DRIVER_OUTPUT_CONTROL, dataseq...)

	// self.send_command(BOOSTER_SOFT_START_CONTROL)
	// self.send_data(0xD7)
	// self.send_data(0xD6)
	// self.send_data(0x9D)
	e.CallFunction(BOOSTER_SOFT_START_CONTROL, 0xD7, 0xD6, 0x9D)

	// self.send_command(WRITE_VCOM_REGISTER)
	// self.send_data(0xA8)                     # VCOM 7C
	e.CallFunction(WRITE_VCOM_REGISTER, 0xA8)

	// self.send_command(SET_DUMMY_LINE_PERIOD)
	// self.send_data(0x1A)                     # 4 dummy lines per gate
	e.CallFunction(SET_DUMMY_LINE_PERIOD, 0x1A)

	// self.send_command(SET_GATE_TIME)
	// self.send_data(0x08)                     # 2us per line
	e.CallFunction(SET_GATE_TIME, 0x08)

	// self.send_command(DATA_ENTRY_MODE_SETTING)
	// self.send_data(0x03)                     # X increment Y increment
	e.CallFunction(DATA_ENTRY_MODE_SETTING, 0x03)

	e.setLookupTable(e.lutFull)

}

//reset - module reset.often used to awaken the module in deep sleep,
func (e *EPD) reset() {
	embd.DigitalWrite(RST_PIN, embd.Low)
	time.Sleep(200)
	embd.DigitalWrite(RST_PIN, embd.High)
	time.Sleep(200)

}

//
//   @brief: set the look-up table register
func (e *EPD) setLookupTable(full bool) {
	e.lutFull = full

	if e.lutFull {
		e.CallFunction(WRITE_LUT_REGISTER, e.lutFullUpdate...)
	} else {
		e.CallFunction(WRITE_LUT_REGISTER, e.lutPartialUpdate...)
	}

}

// Ensure to wait before any next command is executed.. monitors the
// BUSY_PIN
func (e *EPD) wait() {
	var busy int
	var err error
	for ; busy == 1; busy, err = embd.DigitalRead(BUSY_PIN) {
		if err != nil {
			log.Panic("Error waiting BUSY_PIN", err)
		}
		time.Sleep(100) // polling for every 100ms
	}

}

// wait_until_idle(self):
//         while(self.digital_read(self.busy_pin) == 1):      # 0: idle, 1: busy
//             self.delay_ms(100)

func (e *EPD) Sleep(full bool) {
	e.CallFunction(DEEP_SLEEP_MODE)
	e.wait()
	//  self.send_command(DEEP_SLEEP_MODE)
}

// ##
//  #  @brief: update the display
//  #          there are 2 memory areas embedded in the e-paper display
//  #          but once this function is called,
//  #          the the next action of SetFrameMemory or ClearFrame will
//  #          set the other memory area.
//  ##
func (e *EPD) DisplayFrame() {
	e.CallFunction(DISPLAY_UPDATE_CONTROL_2, 0xC4)
	e.CallFunction(MASTER_ACTIVATION)
	e.CallFunction(TERMINATE_FRAME_READ_WRITE)
	e.wait()
}

// ##
//  #  @brief: specify the memory area for data R/W
// def set_memory_area(self, x_start, y_start, x_end, y_end)
func (e *EPD) setMemArea(x0, y0, x1, y1 byte) {
	//   x point must be the multiple of 8 or the last 3 bits will be ignored
	e.CallFunction(SET_RAM_X_ADDRESS_START_END_POSITION, (x0>>3)&0xFF, (x1>>3)&0xFF)
	e.CallFunction(SET_RAM_Y_ADDRESS_START_END_POSITION, y0&0xFF, (y0>>8)&0xFF, y1&0xFF, (y1>>8)&0xFF)
}

/*
   @brief: specify the start point for data R/W in the memory
   //set_memory_pointer()
*/
func (e *EPD) SetXY(x, y byte) {
	e.CallFunction(SET_RAM_X_ADDRESS_COUNTER, (x>>3)&0xFF)
	e.CallFunction(SET_RAM_Y_ADDRESS_COUNTER, y&0xFF, (y>>8)&0xFF)
	e.wait()
}

// #
//  #  @brief: clear the frame memory with the specified color.
//  #          this won't update the display.
func (e *EPD) ClearFrame(color byte) {
	e.setMemArea(0, 0, EPD_WIDTH-1, EPD_HEIGHT-1)
	e.SetXY(0, 0)
	e.CallFunction(WRITE_RAM)

	L := int((EPD_WIDTH / 8) * EPD_HEIGHT) // 8pixels cols = 1 byte
	for i := 0; i < L; i++ {
		e.SendData(color)
	}
}

// ##
//  #  @brief: convert an image to a buffer
//  ## Generates a Byte Buffer
// def get_frame_buffer(self, image):
func (e *EPD) GetFrame() *image.Gray {
	img := image.NewGray(image.Rect(0, 0, int(EPD_WIDTH), int(EPD_HEIGHT)))

	return img
}

// buf = [0x00] * (self.width * self.height / 8)
// # Set buffer to value of Python Imaging Library image.
// # Image must be in mode 1.
// image_monocolor = image.convert('1')
// imwidth, imheight = image_monocolor.size
// if imwidth != self.width or imheight != self.height:
//     raise ValueError('Image must be same dimensions as display \
//         ({0}x{1}).' .format(self.width, self.height))

// pixels = image_monocolor.load()
// for y in range(self.height):
//     for x in range(self.width):
//         # Set the bits for the column of pixels at the current position.
//         if pixels[x, y] != 0:
//             buf[(x + y * self.width) / 8] |= 0x80 >> (x % 8)
// return buf

//  ##
//  #  @brief: put an (SUB) image to the frame memory.
//  #          this won't update the display.
//  ##    def set_frame_memory(self, image, x, y):
func (e *EPD) SetFrame(img image.Gray, x0, y0 byte) {
	w, h := byte(img.Bounds().Dx()), byte(img.Bounds().Dy())

	var x1, y1 byte
	// if (image == None or x < 0 or y < 0):
	//     return
	// image_monocolor = image.convert('1')
	// image_width, image_height  = image_monocolor.size
	// # x point must be the multiple of 8 or the last 3 bits will be ignored
	x0 = x0 & 0xF8
	// image_width = image_width & 0xF8

	// if (x + image_width >= self.width):
	//     x_end = self.width - 1
	// else:
	//     x_end = x + image_width - 1
	// if (y + image_height >= self.height):
	//     y_end = self.height - 1
	// else:
	//     y_end = y + image_height - 1

	x1 = x0 + (w) - 1
	y1 = y0 + (h) - 1
	if x0+w >= EPD_WIDTH {
		x1 = EPD_WIDTH - 1
	}
	if y0+h >= EPD_HEIGHT {
		y1 = EPD_HEIGHT - 1
	}

	// self.set_memory_area(x, y, x_end, y_end)
	// self.set_memory_pointer(x, y)
	// self.send_command(WRITE_RAM)
	e.setMemArea(x0, y0, x1, y1)
	e.SetXY(x0, y0)
	e.SendCommand(WRITE_RAM)
	// # send the image data

	// pixels = image_monocolor.load()
	// byte_to_send = 0x00
	// for j in range(0, y_end - y + 1):
	//     // # 1 byte = 8 pixels, steps of i = 8
	//     for i in range(0, x_end - x + 1):
	//         // # Set the bits for the column of pixels at the current position.
	//         if pixels[i, j] != 0:
	//             byte_to_send |= 0x80 >> (i % 8)
	//         if (i % 8 == 7):
	//             self.send_data(byte_to_send)
	// 			byte_to_send = 0x00
	rr := int(y1 - y0 + 1)
	cc := int(x1 - x0 + 1)
	for row := 0; row < rr; row++ {
		for col := 0; col < cc; col = col + 8 {
			// pixel := img.At(row, col)
			pixel := img.GrayAt(row, col).Y
			bytepix := byte(pixel)
			// if pixel != 0 {
			// 	byte_to_send |= 0x80 >> (uint8(col) % 8)
			// }
			// if col%8 == 7 {
			// e.SendData(byte_to_send)
			// }
			// byte_to_send = 0x00
			e.SendData(bytepix)

		}
	}

}

//Image2Byte assumes binary image of size R*C = R*(C/8)
func Mono2ByteImage(img *image.Gray) (byteimg image.Gray) {

	R := img.Rect.Dy()
	C := img.Rect.Dy()
	CC := C / 8 // 8pixels per byte
	glog.Infoln("Image2Byte (pixw,byte) ", C, "  :  ", CC)

	epdimg := image.NewGray(image.Rect(0, 0, R, CC))
	var cg color.Gray
	var bitstr string
	for r := 0; r < R; r++ {
		bc := 0
		for c := 0; c < C; c++ {
			pix := img.GrayAt(r, c).Y
			if pix > 0 { // 0 if monochrome or 128 if gray scale
				bitstr += "1"
			} else {
				bitstr += "0"
			}

			if len(bitstr) == 8 {
				val, e := strconv.ParseUint(bitstr, 2, 8)
				logme("Image2Byte", e)

				cg.Y = byte(val)
				epdimg.SetGray(r, bc, cg)
				bc++
				bitstr = ""
			}
		}
	}
	return *epdimg
}

func logme(info string, e error) {
	if e != nil {
		log.Panicln(info, " : ", e)
	}
}
