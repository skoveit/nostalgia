#include <windows.h>
#include <stdio.h>

typedef HMODULE (WINAPI *pGMHA)(LPCSTR lpModuleName);
typedef FARPROC (WINAPI *pGPA)(HMODULE hModule, LPCSTR lpProcName);
typedef BOOL (WINAPI *pVP)(LPVOID lpAddress, SIZE_T dwSize, DWORD flNewProtect, PDWORD lpflOldProtect);
typedef VOID (WINAPI *pSleep)(DWORD dwMilliseconds);

const char KRNL[] = { 0x4d, 0x56, 0x5e, 0x4f, 0x56, 0x59, 0x33, 0x3d, 0x31, 0x39, 0x31, 0x00 };
const char VP_STR[] = { 0x56, 0x5c, 0x49, 0x57, 0x57, 0x49, 0x43, 0x52, 0x57, 0x55, 0x54, 0x43, 0x00 };
const char GPA_STR[] = { 0x47, 0x4c, 0x51, 0x54, 0x58, 0x52, 0x64, 0x6e, 0x67, 0x50, 0x55, 0x53, 0x55, 0x00 };
const char GMHA_STR[] = { 0x47, 0x4c, 0x51, 0x54, 0x58, 0x52, 0x4d, 0x50, 0x59, 0x58, 0x56, 0x4c, 0x45, 0x00 };
const char SLP_STR[] = { 0x53, 0x5d, 0x56, 0x56, 0x51, 0x00 };
const char HELLO_STR[] = { 0x68, 0x6c, 0x6c, 0x76, 0x24, 0x24, 0x70, 0x74, 0x79, 0x23, 0x24, 0x23, 0x6a, 0x61, 0x6a, 0x64, 0x6b, 0x00 };

#define XOR_KEY 0x30

VOID Obf(char *d) {
    char k = XOR_KEY;
    char *c = d;
    while (*c) {
        *c = *c ^ k;
        c++;
    }
}

VOID PrintMsg(char *m) {
    Obf(m);
    printf("%s", m);
    Obf(m);
}

void CryptBuffer(PVOID addr, SIZE_T sz) {
    UCHAR *b = (UCHAR *)addr;
    SIZE_T i = 0;
    for (i = 0; i < sz; i++) {
        b[i] ^= XOR_KEY;
    }
}

void DormantCryptor(PVOID cB, SIZE_T sz) {
    Obf(KRNL);
    HMODULE hKrn = LoadLibraryA(KRNL);
    Obf(KRNL);

    Obf(GPA_STR);
    pGPA GetProcAddr = (pGPA)GetProcAddress(hKrn, GPA_STR);
    Obf(GPA_STR);

    Obf(VP_STR);
    pVP VirtualProtect_ptr = (pVP)GetProcAddr(hKrn, VP_STR);
    Obf(VP_STR);

    Obf(SLP_STR);
    pSleep Sleep_ptr = (pSleep)GetProcAddr(hKrn, SLP_STR);
    Obf(SLP_STR);

    DWORD oldP;
    if (VirtualProtect_ptr(cB, sz, PAGE_EXECUTE_READWRITE, &oldP)) {
        CryptBuffer(cB, sz);
        PrintMsg((char*)HELLO_STR);
        Sleep_ptr(10000);
        CryptBuffer(cB, sz);
        VirtualProtect_ptr(cB, sz, oldP, &oldP);
    }
}

void PayloadEntry() {
    char h[] = { 0x68, 0x6c, 0x6c, 0x76, 0x24, 0x24, 0x70, 0x74, 0x79, 0x23, 0x24, 0x23, 0x6a, 0x61, 0x6a, 0x64, 0x6b, 0x0a, 0x00 };
    PrintMsg(h);
    
    char *s = (char*)malloc(1024);
    if (s) {
        for(int i = 0; i < 1023; i++) s[i] = (char)(i % 26 + 'a');
        s[1023] = 0;
        printf("Allocated: %p\n", (PVOID)s);
        PrintMsg((char*)HELLO_STR);
        free(s);
    } else {
        printf("Memory allocation failed.\n");
    }
}

int main(int argc, char **argv) {
    char m[] = { 0x48, 0x4d, 0x4c, 0x4c, 0x23, 0x24, 0x22, 0x70, 0x74, 0x79, 0x23, 0x24, 0x23, 0x6a, 0x61, 0x6a, 0x64, 0x6b, 0x0a, 0x00 };
    PrintMsg(m);

    ULONG_PTR startAddr = (ULONG_PTR)PayloadEntry;
    ULONG_PTR endAddr = (ULONG_PTR)DormantCryptor;
    SIZE_T codeSize = endAddr - startAddr;

    DormantCryptor((PVOID)startAddr, codeSize);

    PayloadEntry();

    char f[] = { 0x46, 0x4d, 0x4e, 0x4c, 0x24, 0x24, 0x70, 0x74, 0x79, 0x23, 0x24, 0x23, 0x6a, 0x61, 0x6a, 0x64, 0x6b, 0x0a, 0x00 };
    PrintMsg(f);

    return 0;
}