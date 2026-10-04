#ifdef FEATURE_FLAG
_Bool enabled(_Bool flag) {
    if (flag) { return 1; }
    return 0;
}
#else
_Bool enabled(_Bool flag) { return flag; }
#endif
