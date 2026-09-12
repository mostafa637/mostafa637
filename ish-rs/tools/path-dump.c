// Reference generator for path_is_normalized and path_next_component
// Uses unmodified fs/path.c but stubs out mount/fs dependencies for the leaf helpers.

#include <stdio.h>
#include <string.h>
#include <stdbool.h>
#include <stdlib.h>

// We copy the leaf functions directly to avoid linking the whole fs layer.
// This still tests against the exact C logic.

bool path_is_normalized(const char *path) {
    while (*path != '\0') {
        if (*path != '/')
            return false;
        path++;
        if (*path == '/')
            return false;
        while (*path != '/' && *path != '\0')
            path++;
    }
    return true;
}

#define MAX_NAME 256
bool path_next_component(const char **path, char *component, int *err) {
    const char *p = *path;
    if (*p == '\0')
        return false;
    if (*p != '/') abort();
    p++;
    char *c = component;
    while (*p != '/' && *p != '\0') {
        *c++ = *p++;
        if (c - component >= MAX_NAME) {
            *err = 36;
            return false;
        }
    }
    *c = '\0';
    *path = p;
    return true;
}

int main() {
    const char *cases[] = {
        "",
        "/",
        "/a",
        "/a/b",
        "/a/b/c",
        "a",
        "//",
        "/a//b",
        "/a/",
        "/a/b/",
        "/a//",
        NULL
    };
    for (int i = 0; cases[i]; i++) {
        printf("NORM \"%s\" -> %d\n", cases[i], path_is_normalized(cases[i]));
    }

    const char *path_cases[] = {
        "/a/bb/ccc",
        "/",
        "/single",
        "/a/b/c/d",
        "",
        NULL
    };
    for (int i = 0; path_cases[i]; i++) {
        const char *p = path_cases[i];
        printf("ITER \"%s\":\n", p);
        char comp[MAX_NAME];
        int err = 0;
        while (path_next_component(&p, comp, &err)) {
            printf("  COMP \"%s\" remaining=\"%s\"\n", comp, p);
        }
        if (err) printf("  ERR %d\n", err);
        else printf("  END remaining=\"%s\"\n", p);
    }

    return 0;
}
