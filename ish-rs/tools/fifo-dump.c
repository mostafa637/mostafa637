// Reference generator for fifo.h/.c differential test.
// Links unmodified util/fifo.c and drives operations.

#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include "util/fifo.h"

static void dump_fifo(struct fifo *f, const char *label) {
    printf("S %s cap=%zu size=%zu start=%zu data=", label, fifo_capacity(f), fifo_size(f), f->start);
    for (size_t i = 0; i < f->capacity; i++) {
        printf("%02x", (unsigned char)f->buf[i]);
    }
    printf(" snapshot=");
    for (size_t i = 0; i < f->size; i++) {
        size_t idx = (f->start + i) % f->capacity;
        printf("%02x", (unsigned char)f->buf[idx]);
    }
    printf("\n");
}

int main() {
    struct fifo f;
    fifo_init(&f, 8);
    printf("O INIT cap=8\n");
    dump_fifo(&f, "after_init");

    const char *a = "abc";
    int r = fifo_write(&f, a, 3, 0);
    printf("O WRITE 3 abc R %d\n", r);
    dump_fifo(&f, "after_write_abc");

    char out[3];
    r = fifo_read(&f, out, 3, 0);
    printf("O READ 3 R %d data=", r);
    for (int i = 0; i < 3; i++) printf("%02x", (unsigned char)out[i]);
    printf("\n");
    dump_fifo(&f, "after_read_abc");

    // Overwrite test
    fifo_flush(&f);
    printf("O FLUSH\n");
    dump_fifo(&f, "after_flush");
    fifo_write(&f, "abcd", 4, 0);
    printf("O WRITE 4 abcd R 0\n");
    dump_fifo(&f, "after_abcd");
    r = fifo_write(&f, "e", 1, 0);
    printf("O WRITE 1 e no_overwrite R %d\n", r);
    dump_fifo(&f, "after_e_no_overwrite");
    r = fifo_write(&f, "e", 1, FIFO_OVERWRITE);
    printf("O WRITE 1 e overwrite R %d\n", r);
    dump_fifo(&f, "after_e_overwrite");

    // Peek and LAST
    fifo_flush(&f);
    printf("O FLUSH\n");
    dump_fifo(&f, "after_flush2");
    fifo_write(&f, "012345", 6, 0);
    printf("O WRITE 6 012345 R 0\n");
    dump_fifo(&f, "after_012345");

    char out2[2];
    r = fifo_read(&f, out2, 2, FIFO_PEEK);
    printf("O READ 2 PEEK R %d data=", r);
    for (int i = 0; i < 2; i++) printf("%02x", (unsigned char)out2[i]);
    printf("\n");
    dump_fifo(&f, "after_peek_2");

    r = fifo_read(&f, out2, 2, FIFO_LAST | FIFO_PEEK);
    printf("O READ 2 LAST|PEEK R %d data=", r);
    for (int i = 0; i < 2; i++) printf("%02x", (unsigned char)out2[i]);
    printf("\n");
    dump_fifo(&f, "after_last_peek_2");

    r = fifo_read(&f, out2, 2, FIFO_LAST);
    printf("O READ 2 LAST R %d data=", r);
    for (int i = 0; i < 2; i++) printf("%02x", (unsigned char)out2[i]);
    printf("\n");
    dump_fifo(&f, "after_last_2");

    // Wrap test
    fifo_flush(&f);
    printf("O FLUSH\n");
    dump_fifo(&f, "after_flush3");
    fifo_write(&f, "abcdefgh", 8, 0);
    char tmp[4];
    fifo_read(&f, tmp, 4, 0);
    fifo_write(&f, "12", 2, 0);
    printf("O WRAP test\n");
    dump_fifo(&f, "after_wrap");

    char out4[4];
    r = fifo_read(&f, out4, 4, FIFO_LAST | FIFO_PEEK);
    printf("O READ 4 LAST|PEEK wrap R %d data=", r);
    for (int i = 0; i < 4; i++) printf("%02x", (unsigned char)out4[i]);
    printf("\n");
    dump_fifo(&f, "after_wrap_last_peek");

    fifo_destroy(&f);
    return 0;
}
