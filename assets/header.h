#define __int64 long long
#define __iamcu__
#define __int32 int
#define NTDDI_WIN7 0x06010000
#define __forceinline __attribute__((always_inline))
#define _AMD64_
#define _M_AMD64
#define __unaligned
#define _MSC_FULL_VER 192930133

#define XSTR(x) STR(x)
#define STR(x) #x


#include<sal.h>

#if defined(_In_)
#undef _In_
#define _In_  __attribute__((anno("_In_")))
#endif

#if defined(_In_opt_)
#undef _In_opt_
#define _In_opt_  __attribute__((anno("_In_opt_")))
#endif


#if defined(_Inout_)
#undef _Inout_
#define _Inout_  __attribute__((anno("_Inout_")))
#endif


#if defined(_Out_)
#undef _Out_
#define _Out_  __attribute__((anno("_Out_")))
#endif

#if defined(_Outptr_)
#undef _Outptr_
#define _Outptr_  __attribute__((anno("_Outptr_")))
#endif

#if defined(_Out_opt_)
#undef _Out_opt_
#define _Out_opt_  __attribute__((anno("_Out_opt_")))
#endif

#include<windows.h>
// #include <minwindef.h>
// #include <minwinbase.h>
// #include <winnt.h>

