#include <inttypes.h>
#include <stddef.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

/* 1. Estructura intercalada (1B con 8B, 4B y 2B) */
struct CDR_Desalineado {
    char tipo_red;          /* 1 byte */
    uint64_t imsi_abonado;  /* 8 bytes */
    char bandera_roaming;   /* 1 byte */
    uint32_t celda_id;      /* 4 bytes */
    char codigo_estado;     /* 1 byte */
    uint16_t puerto_origen; /* 2 bytes */
};

/* 2. Estructura ordenada de mayor a menor alineacion natural (64 bits) */
struct CDR_Optimizado {
    uint64_t imsi_abonado;  /* 8 bytes */
    uint32_t celda_id;      /* 4 bytes */
    uint16_t puerto_origen; /* 2 bytes */
    char tipo_red;          /* 1 byte */
    char bandera_roaming;   /* 1 byte */
    char codigo_estado;     /* 1 byte */
};

/* Variable global inicializada en segmento .data */
static struct CDR_Optimizado registro_global_data = {
    .imsi_abonado = 734010012345678ULL,
    .celda_id = 4096U,
    .puerto_origen = 5060U,
    .tipo_red = '5',
    .bandera_roaming = '0',
    .codigo_estado = 'A'
};

int main(void) {
    /* Variable local en el Stack */
    struct CDR_Desalineado registro_stack = {
        .tipo_red = '5',
        .imsi_abonado = 734010012345678ULL,
        .bandera_roaming = '1',
        .celda_id = 8192U,
        .codigo_estado = 'O',
        .puerto_origen = 2152U
    };

    /* Bloque dinamico en el Heap */
    struct CDR_Optimizado *registro_heap =
        (struct CDR_Optimizado *)malloc(sizeof(*registro_heap));
    if (registro_heap == NULL) {
        fprintf(stderr, "Error al asignar memoria en el Heap.\n");
        return EXIT_FAILURE;
    }
    *registro_heap = registro_global_data;

    printf("=== (a) DIRECCIONES VIRTUALES DE MEMORIA ===\n");
    printf("Global (.data) : %p\n", (const void *)&registro_global_data);
    printf("Local  (Stack) : %p\n", (const void *)&registro_stack);
    printf("Dinamica(Heap) : %p\n\n", (const void *)registro_heap);

    size_t tam_desalineado = sizeof(struct CDR_Desalineado);
    size_t tam_optimizado = sizeof(struct CDR_Optimizado);

    printf("=== (b) TAMANO Y DESPLAZAMIENTO (OFFSETOF) ===\n");
    printf("struct CDR_Desalineado -> sizeof = %zu bytes\n", tam_desalineado);
    printf("  offset tipo_red       : %zu\n", offsetof(struct CDR_Desalineado, tipo_red));
    printf("  offset imsi_abonado   : %zu\n", offsetof(struct CDR_Desalineado, imsi_abonado));
    printf("  offset bandera_roaming: %zu\n", offsetof(struct CDR_Desalineado, bandera_roaming));
    printf("  offset celda_id       : %zu\n", offsetof(struct CDR_Desalineado, celda_id));
    printf("  offset codigo_estado  : %zu\n", offsetof(struct CDR_Desalineado, codigo_estado));
    printf("  offset puerto_origen  : %zu\n\n", offsetof(struct CDR_Desalineado, puerto_origen));

    printf("struct CDR_Optimizado -> sizeof = %zu bytes\n", tam_optimizado);
    printf("  offset imsi_abonado   : %zu\n", offsetof(struct CDR_Optimizado, imsi_abonado));
    printf("  offset celda_id       : %zu\n", offsetof(struct CDR_Optimizado, celda_id));
    printf("  offset puerto_origen  : %zu\n", offsetof(struct CDR_Optimizado, puerto_origen));
    printf("  offset tipo_red       : %zu\n", offsetof(struct CDR_Optimizado, tipo_red));
    printf("  offset bandera_roaming: %zu\n", offsetof(struct CDR_Optimizado, bandera_roaming));
    printf("  offset codigo_estado  : %zu\n\n", offsetof(struct CDR_Optimizado, codigo_estado));

    /* (c) Ahorro en 10,000,000 de registros y ocupacion en linea L1 de 64 bytes */
    const double lote = 10000000.0;
    double bytes_desalineado = lote * (double)tam_desalineado;
    double bytes_optimizado = lote * (double)tam_optimizado;
    double millones_bytes_ahorrados = (bytes_desalineado - bytes_optimizado) / 1000000.0;

    size_t cache_line_l1 = 64;
    size_t registros_l1_desalineado = cache_line_l1 / tam_desalineado;
    size_t registros_l1_optimizado = cache_line_l1 / tam_optimizado;

    printf("=== (c) LOTE DE 10,000,000 REGISTROS Y LINEA CACHE L1 (64B) ===\n");
    printf("Total lote Desalineado : %.0f bytes (%.2f millones de bytes)\n",
           bytes_desalineado, bytes_desalineado / 1000000.0);
    printf("Total lote Optimizado  : %.0f bytes (%.2f millones de bytes)\n",
           bytes_optimizado, bytes_optimizado / 1000000.0);
    printf("RAM ahorrada en lote   : %.2f millones de bytes\n",
           millones_bytes_ahorrados);
    printf("Registros por linea L1 (64B) - Desalineado : %zu\n",
           registros_l1_desalineado);
    printf("Registros por linea L1 (64B) - Optimizado  : %zu\n",
           registros_l1_optimizado);

    /* Liberar memoria dinamica sin fugas */
    free(registro_heap);
    registro_heap = NULL;

    return EXIT_SUCCESS;
}