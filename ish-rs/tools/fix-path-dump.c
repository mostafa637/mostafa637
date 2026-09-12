// Reference generator for fix_path differential test.
// Links unmodified fs/fix_path.h (header-only) and prints test cases.

#include <stdio.h>
#include <string.h>
#include "fs/fix_path.h"

int main() {
    const char *cases[] = {
        "",
        "/",
        "/foo",
        "/foo/bar",
        "foo",
        "foo/bar",
        "//double",
        "/a//b",
        "/a/./b",
        "/a/../b",
        "a/b/../c",
        "/trailing/",
        NULL
    };

    for (int i = 0; cases[i] != NULL; i++) {
        const char *input = cases[i];
        const char *output = fix_path(input);
        // Print as hex for input and output to handle empty string
        printf("C %zu ", strlen(input));
        for (size_t j = 0; j < strlen(input); j++) {
            printf("%02x", (unsigned char)input[j]);
        }
        printf(" -> %zu ", strlen(output));
        for (size_t j = 0; j < strlen(output); j++) {
            printf("%02x", (unsigned char)output[j]);
        }
        printf(" \"%s\" -> \"%s\"\n", input, output);
    }

    // Byte version
    const unsigned char *bcases[] = {
        (unsigned char *)"",
        (unsigned char *)"/",
        (unsigned char *)"/foo",
        (unsigned char *)"foo",
        NULL
    };
    for (int i = 0; bcases[i] != NULL; i++) {
        const unsigned char *input = bcases[i];
        size_t len = strlen((const char *)input);
        const char *output = fix_path((const char *)input);
        printf("B %zu ", len);
        for (size_t j = 0; j < len; j++) printf("%02x", input[j]);
        printf(" -> %zu ", strlen(output));
        for (size_t j = 0; j < strlen(output); j++) printf("%02x", (unsigned char)output[j]);
        printf("\n");
    }

    return 0;
}
