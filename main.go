package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path"
)

const PNG_SIG_SIZE = 8
var png_sig = []byte{137, 80, 78, 71, 13, 10, 26, 10}

func check(err error) {
    if err != nil {
        panic(err)
    }
}

func usage(program string) {
    panic(fmt.Sprintf("usage: %s <input.png> <output.png>", program))
}

func copy_bytes(from []byte, to *os.File, from_cursor *int, to_cursor *int) (int, error) {
    // slice out the data from the contents
    data := from[*from_cursor:*from_cursor+4]
    *from_cursor += 4

    // copy data from buf to output file
    n, err := to.WriteAt(data, int64(*to_cursor))
    if err != nil {
        return 0, err
    }
    if n != 4 {
        return n, errors.New(fmt.Sprintf("ERROR: written wrong no of bytes to file %s.", to.Name()))
    }
    *to_cursor += n

    return n, nil
}

// CRC Code

var crc_table = make([]uint32, 256)
var crc_table_computed = false

func make_crc_table() {
    for n := 0; n < 256; n++ {
        c := uint32(n)
        for k := 0; k < 8; k++ {
            if c & 1 == 1 {
                c = 0xedb88320 ^ (c >> 1)
            } else {
                c = c >> 1
            }
        }
        crc_table[n] = c
    }
    crc_table_computed = true
}

func update_crc(crc uint32, buf []uint8) uint32 {
    c := crc
    if !crc_table_computed {
        make_crc_table()
    }

    for n := 0; n < len(buf); n++ {
        c = crc_table[(c ^ uint32(buf[n])) & 0xff] ^ (c >> 8)
    }
    return c
}

func crc_func(buf []uint8) uint32 {
    return update_crc(0xffffffff, buf) ^ 0xffffffff
}

// CRC Code end

func main() {
    program := path.Base(os.Args[0])
    if len(os.Args) < 3 {
        usage(program)
    }
    input_file_path := os.Args[1]
    output_file_path := os.Args[2]

    // read the file at once
    input_file_contents, err := os.ReadFile(input_file_path)
    check(err)

    cursor := int64(0)
    output_cursor := int64(0)

    running := true

    // slice out the signature from the contents
    sig := input_file_contents[cursor:cursor+PNG_SIG_SIZE]
    cursor += PNG_SIG_SIZE

    // if signature of file does not match with PNG format then panic
    if !bytes.Equal(sig, png_sig) {
        fmt.Println(sig)
        panic(fmt.Sprintf("ERROR: %s is not a valid png file.", input_file_path))
    }

    // create the output file
    output_file, err := os.Create(output_file_path)
    check(err)
    defer output_file.Close()

    // copy sig from input file to output file
    n, err := output_file.WriteAt(sig, output_cursor)
    check(err)
    if n != PNG_SIG_SIZE {
        panic(fmt.Sprintf("ERROR: could not write to file %s.", output_file_path))
    }
    output_cursor += PNG_SIG_SIZE

    for running {
        // slice out the chunk size from the contents
        chunk_size := input_file_contents[cursor:cursor+4]
        cursor += 4

        // copy chunk_size from input file to output file
        n, err = output_file.WriteAt(chunk_size, output_cursor)
        check(err)
        if n != 4 {
            panic(fmt.Sprintf("ERROR: could not write to file %s.", output_file_path))
        }
        output_cursor += 4

        // slice out the chunk type from the contents
        chunk_type := input_file_contents[cursor:cursor+4]
        cursor += 4

        if bytes.Equal(chunk_type, []byte{73, 69, 78, 68}) {
            running = false
        }

        // copy chunk_type from input file to output file
        n, err = output_file.WriteAt(chunk_type, output_cursor)
        check(err)
        if n != 4 {
            panic(fmt.Sprintf("ERROR: could not write to file %s.", output_file_path))
        }
        output_cursor += 4

        // slice out the chunk data from the contents
        chunk_size_int := int64(binary.BigEndian.Uint32(chunk_size))
        chunk_data := input_file_contents[cursor:cursor+chunk_size_int]
        cursor += chunk_size_int

        // copy chunk daya from input file to output file
        n, err = output_file.WriteAt(chunk_data, output_cursor)
        check(err)
        if n != int(chunk_size_int) {
            panic(fmt.Sprintf("ERROR: could not write to file %s.", output_file_path))
        }
        output_cursor += chunk_size_int

        // slice out crc from the contents
        crc := input_file_contents[cursor:cursor+4]
        cursor += 4

        // copy crc from input file to output file
        n, err = output_file.WriteAt(crc, output_cursor)
        check(err)
        if n != 4 {
            panic(fmt.Sprintf("ERROR: could not write to file %s.", output_file_path))
        }
        output_cursor += 4

        if bytes.Equal(chunk_type, []byte{73, 68, 65, 84}) {
            injected_buf := []byte("YEP")
            injected_size := len(injected_buf)
            injected_size_buf := make([]byte, 4)
            binary.LittleEndian.PutUint32(injected_size_buf, uint32(injected_size))
            output_file.WriteAt(injected_buf, output_cursor)
            injected_type := []byte("coCK")
            injected_crc := crc_func(injected_buf)
            injected_crc_buf := make([]byte, 4)
            binary.LittleEndian.PutUint32(injected_crc_buf, injected_crc)

            output_file.WriteAt(injected_size_buf, output_cursor)
            output_cursor += 4

            output_file.WriteAt(injected_type, output_cursor)
            output_cursor += 4

            output_file.WriteAt(injected_buf, output_cursor)
            output_cursor += int64(len(injected_buf))

            output_file.WriteAt(injected_crc_buf, output_cursor)
            output_cursor += 4
        }
    }
}
