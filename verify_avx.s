.text
.global verify_avx
verify_avx:
    vpaddd %ymm2, %ymm3, %ymm1
    vaddps %ymm2, %ymm3, %ymm1
    vpxor %ymm0, %ymm0, %ymm0
    vpshufd $1, %ymm2, %ymm1
    vblendps $1, %ymm2, %ymm3, %ymm1
    vblendpd $2, %ymm2, %ymm3, %ymm1
    vpcmpeqb %ymm2, %ymm3, %ymm1
    vpsllw $1, %ymm2, %ymm1
    vpsrld $2, %ymm2, %ymm1
    vpsrad $1, %ymm2, %ymm1
    vpsllq $3, %ymm2, %ymm1
    vpsrlq $3, %ymm2, %ymm1
    vpslldq $4, %ymm2, %ymm1
    vpsrldq $4, %ymm2, %ymm1
    vpsllvd %ymm2, %ymm3, %ymm1
    vpsrlvd %ymm2, %ymm3, %ymm1
    vpsravd %ymm2, %ymm3, %ymm1
    vpsllvq %ymm2, %ymm3, %ymm1
    vpsrlvq %ymm2, %ymm3, %ymm1
    ret
