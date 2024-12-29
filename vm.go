package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

const MEMORY_MAX int = 1<<16
var memory [MEMORY_MAX]uint16

const (
	R_R0 = iota
	R_R1
	R_R2
	R_R3
	R_R4
	R_R5
	R_R6
	R_R7
	R_PC
	R_COND
	R_COUNT
)
var reg [R_COUNT]uint16

const (
	OP_BR = iota /* branch */
	OP_ADD       /* add  */
	OP_LD        /* load */
	OP_ST        /* store */
	OP_JSR       /* jump register */
	OP_AND       /* bitwise and */
	OP_LDR       /* load register */
	OP_STR       /* store register */
	OP_RTI       /* unused */
	OP_NOT       /* bitwise not */
	OP_LDI       /* load indirect */
	OP_STI       /* store indirect */
	OP_JMP       /* jump */
	OP_RES       /* reserved (unused) */
	OP_LEA       /* load effective address */
	OP_TRAP      /* execute trap */
)

const (
	FL_POS = 1 << 0 /* P */
	FL_ZRO = 1 << 1 /* Z */
	FL_NEG = 1 << 2 /* N */
)

const (
	TRAP_GETC = 0x20  /* get character from keyboard, not echoed onto the terminal */
	TRAP_OUT = 0x21   /* output a character */
	TRAP_PUTS = 0x22  /* output a word string */
	TRAP_IN = 0x23    /* get character from keyboard, echoed onto the terminal */
	TRAP_PUTSP = 0x24 /* output a byte string */
	TRAP_HALT = 0x25  /* halt the program */
)

const (
	MR_KBSR = 0xFE00  /* keyboard status */
	MR_KBDR = 0xFE02  /* keyboard data */
)

func memRead(addr uint16) uint16 {
	if addr == MR_KBSR {
		if checkKey() {
			b := make([]byte,1)
			os.Stdin.Read(b)
			memory[MR_KBSR] = 1 << 15
			memory[MR_KBDR] = uint16(b[0])
		} else {
			memory[MR_KBSR] = 0
		}
	}
	return memory[addr]
}

func memWrite(addr uint16, val uint16) {
	memory[addr] = val
}

func checkKey() bool {
	panic("hello?")
	return true
}

func readImage(path string) {
	file, err := os.Open(path)
	if err != nil {
		panic("Cannot open a file")
	}
	bytes := make([]byte, MEMORY_MAX)
	_, err2 := file.Read(bytes)
	if err2 != nil {
		panic("Cannot read file")
	}
	origin := binary.BigEndian.Uint16(bytes[:2])
	for i := 2; i < len(bytes); i += 2 {
		memory[origin] = binary.BigEndian.Uint16(bytes[i:i+2])
		origin++
	}
}

func signExtend(x uint16, bit_count int) uint16 {
	if (x >> (bit_count - 1)) & 1 > 0 {
		x |= (0xFFFF << bit_count)
	}
	return x
}

func updateFlags(r uint16) {
	if reg[r] == 0 {
		reg[R_COND] = FL_ZRO
	} else if (reg[r] >> 15) > 0 {
		reg[R_COND] = FL_NEG
	} else {
		reg[R_COND] = FL_POS
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "LC3 [image-file1] ...")
		os.Exit(1)	
	}
	for i,v := range os.Args {
		if i == 0 { continue }
		readImage(v)
	}
	reg[R_COND] = FL_ZRO
	const (
		PC_START = 0x3000 /* Set the PC starting position. 0x3000 is the default*/
	)
	reg[R_PC] = PC_START

	running := true
	for running {
		instr := memRead(reg[R_PC]); reg[R_PC] += 1
		op := instr >> 12
		switch op {
		case OP_ADD:
			r0 := (instr >> 9) & 0x7
			r1 := (instr >> 6) & 0x7
			imm_flag := (instr >> 5) & 0x1
			if imm_flag == 1 {
				imm5 := signExtend(instr & 0x1F, 5)
				reg[r0] = reg[r1] + imm5
			} else {
				r2 := instr & 0x5
				reg[r0] = reg[r1] + reg[r2]
			}
			updateFlags(r0)
		case OP_AND:
			r0 := (instr >> 9) & 0x7
			r1 := (instr >> 6) & 0x7
			imm_flag := (instr >> 5) & 0x1
			if imm_flag > 0 {
				imm5 := signExtend(instr & 0x1F, 5)
				reg[r0] = reg[r1] & imm5
			} else {
				r2 := instr & 0x7
				reg[r0] = reg[r1] & reg[r2]
			}
			updateFlags(r0)
		case OP_NOT:
			r0 := (instr >> 9) & 0x7
			r1 := (instr >> 6) & 0x7
			reg[r0] = ^reg[r1]
			updateFlags(r0)
		case OP_BR:
			pc_offset := signExtend(instr & 0x1FF, 9)
			cond_flag := (instr >> 9) & 0x7
			if (cond_flag & reg[R_COND]) > 0 {
				reg[R_PC] += pc_offset
			}
		case OP_JMP:
			r1 := (instr >> 6) & 0x7
			reg[R_PC] = r1
		case OP_JSR:
			long_flag := (instr >> 11) & 1
			reg[R_R7] = reg[R_PC]
			if long_flag > 0 {
				long_pc_offset := signExtend(instr & 0x7FF, 11)
				reg[R_PC] += long_pc_offset
			} else {
				r1 := (instr >> 6) & 0x7
				reg[R_PC] = reg[r1]
			}
		case OP_LD:
			r0 := (instr >> 9) & 0x7
			pc_offset := signExtend(instr & 0x1FF, 9)
			reg[r0] = memRead(reg[R_PC] + pc_offset)
			updateFlags(r0)
		case OP_LDI:
			r0 := (instr >> 9) & 0x7
			pc_offset := signExtend(instr & 0x1FF, 9)
			reg[r0] = memRead(reg[R_PC] + pc_offset)
			updateFlags(r0)
		case OP_LDR:
			r0 := (instr >> 9) & 0x7
			r1 := (instr >> 6) & 0x7
			offset := signExtend(instr & 0x3F, 6)
			reg[r0] = memRead(reg[r1] + offset)
			updateFlags(r0)
		case OP_LEA:
			r0 := (instr >> 9) & 0x7
			pc_offset := signExtend(instr & 0x1FF, 9)
			reg[r0] = reg[R_PC] + pc_offset
			updateFlags(r0)
		case OP_ST:
			r0 := (instr >> 9) & 0x7
			pc_offset := signExtend(instr & 0x1FF, 9)
			memWrite(reg[R_PC] + pc_offset, reg[r0])
		case OP_STI:
			r0 := (instr >> 9) & 0x7
			pc_offset := signExtend(instr & 0x1FF, 9)
			memWrite(memRead(reg[R_PC] + pc_offset), reg[r0])
		case OP_STR:
			r0 := (instr >> 9) & 0x7
			r1 := (instr >> 6) & 0x7
			offset := signExtend(instr & 0x3F, 6)
			memWrite(reg[r1] + offset, reg[r0])
		case OP_TRAP:
			reg[R_R7] = reg[R_PC]
			switch instr & 0xFF {
			case TRAP_GETC:
				var b byte 
				fmt.Scanf("%c", &b)
				reg[R_R0] = uint16(b)
				updateFlags(R_R0)
			case TRAP_OUT:
				fmt.Print(string(rune(reg[R_R0])))
			case TRAP_PUTS:
				index := reg[R_R0]
				c := memory[index]
				for c > 0 {
					s := string(rune(c))
					fmt.Print(s)
					index += 1
					c = memory[index]
				}
			case TRAP_IN:
				fmt.Printf("Enter a character: ")
				var b byte 
				fmt.Print(string(b))
				reg[R_R0] = uint16(b)
				updateFlags(R_R0)
			case TRAP_PUTSP:
				index := reg[R_R0]
				c := memory[index]
				for c > 0 {
					char1 := c & 0xFF
					s := string(rune(char1))
					fmt.Print(s)
					char2 := c >> 8
					s = string(rune(char2))
					fmt.Print(s)
					index += 1
					c = memory[index]
				}
			case TRAP_HALT:
				fmt.Println("HALT")
				running = false
			}
		//case OP_RES:
		//case OP_RTI:
		default:
			panic("Bad Opcode")
		}
	}
}