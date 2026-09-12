// Reference generator for bits.h differential test.
#include <stdio.h>
#include <string.h>
#include <stdbool.h>
#include "util/bits.h"

int main() {
    printf("SIZE 0 -> %d\n", (int)BITS_SIZE(0));
    printf("SIZE 1 -> %d\n", (int)BITS_SIZE(1));
    printf("SIZE 8 -> %d\n", (int)BITS_SIZE(8));
    printf("SIZE 9 -> %d\n", (int)BITS_SIZE(9));
    printf("SIZE 16 -> %d\n", (int)BITS_SIZE(16));

    unsigned char data[4] = {0};
    bit_set(0, data);
    printf("SET 0 -> %02x %02x %02x %02x test0=%d\n", data[0], data[1], data[2], data[3], bit_test(0, data));
    bit_set(9, data);
    printf("SET 9 -> %02x %02x %02x %02x test9=%d test1=%d\n", data[0], data[1], data[2], data[3], bit_test(9, data), bit_test(1, data));
    bit_clear(0, data);
    printf("CLEAR 0 -> %02x %02x %02x %02x test0=%d test9=%d\n", data[0], data[1], data[2], data[3], bit_test(0, data), bit_test(9, data));
    bit_set(8, data);
    printf("SET 8 -> %02x %02x %02x %02x\n", data[0], data[1], data[2], data[3]);
    bit_clear(8, data);
    printf("CLEAR 8 -> %02x %02x %02x %02x\n", data[0], data[1], data[2], data[3]);

    // Test sequence
    unsigned char seq[2] = {0};
    for (int i = 0; i < 16; i++) {
        if (i % 3 == 0) bit_set(i, seq);
        printf("SEQ %d -> %02x %02x\n", i, seq[0], seq[1]);
    }
    for (int i = 0; i < 16; i++) {
        printf("TEST %d -> %d\n", i, bit_test(i, seq));
    }

    return 0;
}
