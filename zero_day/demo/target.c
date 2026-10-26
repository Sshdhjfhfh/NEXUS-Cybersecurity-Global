/* Deliberately failing, synthetic file parser. No networking or exploitation. */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

int main(int argc, char **argv) {
    if (argc != 2) return 2;
    FILE *input = fopen(argv[1], "rb");
    if (!input) return 2;
    unsigned char data[4096];
    size_t length = fread(data, 1, sizeof(data), input);
    fclose(input);
    /* AFL++ must discover the final byte; the initial seeds do not crash. */
    if (length >= 6 && memcmp(data, "NEXUS!", 6) == 0) abort();
    if (length >= 6 && memcmp(data, "NEXUS?", 6) == 0) return 0;
    return 0;
}
