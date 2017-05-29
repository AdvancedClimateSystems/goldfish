package main

import (
	"flag"
	"fmt"
	"log"

	modbus "github.com/advancedclimatesystems/goldfish"
)

// handleCoils is a handler that responds to Modbus requests with function code
// 1 (read coils) and 2 (read discrete inputs).
//
// The handler is called with 3 parameters: the unit/slave id, the number of
// the first requested address and the total address requested.
//
// The handler must return a slice representing the values of the requested
// addresses like [0, 1, 0, 1, 0, 1].
func handleReadCoils(unitID, start, quantity int) ([]int16, error) {
	coils := make([]int16, quantity)
	for i := 0; i < quantity; i++ {
		coils[i] = int16((i + start) % 2)
	}

	return coils, nil
}

// handleRegisters is a handler that responds to Modbus request with function
// code 3 (read holding registers) and 4 (read input registers).
//
// The handler is called with 3 parameters: the unit/slave id, the number of
// the first requested address and the total address requested.
//
// The handler must return a slice with the values of the registers like
// [31, 298, 1999].
func handleRegisters(unitID, start, quantity int) ([]int16, error) {
	registers := make([]int16, quantity)
	for i := 0; i <= quantity; i++ {
		registers[i] = int16(i)
	}

	return registers, nil
}

func handleWriteRegisters(unitID, start int, values []int16) error {
	return nil
}

func handleWriteCoils(unitID, start int, values []int16) error {
	if start == 1 {
		return modbus.IllegalAddressError
	}
	return nil
}

func main() {
	addr := flag.String("addr", ":502", "address to listen on.")
	flag.Parse()

	s, err := modbus.NewServer(*addr)

	if err != nil {
		log.Fatal(fmt.Sprintf("Failed to start Modbus server: %v", err))
	}

	s.Handle(modbus.ReadCoils, modbus.NewReadHandler(handleReadCoils))
	s.Handle(modbus.ReadHoldingRegisters, modbus.NewReadHandler(handleRegisters))
	s.Handle(modbus.WriteSingleCoil, modbus.NewWriteHandler(handleWriteCoils))
	s.Handle(modbus.WriteSingleRegister, modbus.NewWriteHandler(handleWriteRegisters))

	s.Listen()
}
