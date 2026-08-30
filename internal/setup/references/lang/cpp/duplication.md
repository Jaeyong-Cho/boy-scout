# Duplication Violations — C++ Example

## Problem

Duplicate code increases maintenance cost and makes bugs harder to fix. A change to one copy must be applied consistently across all copies, or subtle behavioral differences emerge.

## Solution

Extract duplicated code into a shared helper function, defined in a header (`.h` / `.hpp`) and implemented in a source file (`.cpp`), then call it from both sites. This follows the DRY (Don't Repeat Yourself) principle.

### Example

**Before: duplicate logic in two files**

```cpp
// file1.cpp
int calculateTotal(int price, int quantity) {
    int subtotal = price * quantity;
    int tax = subtotal * 15 / 100;
    return subtotal + tax;
}

// file2.cpp
int computeAmount(int cost, int count) {
    int subtotal = cost * count;
    int tax = subtotal * 15 / 100;
    return subtotal + tax;
}
```

**After: shared helper function**

```cpp
// math_helper.h
int calculateTotalWithTax(int basePrice, int quantity);

// math_helper.cpp
int calculateTotalWithTax(int basePrice, int quantity) {
    int subtotal = basePrice * quantity;
    int tax = subtotal * 15 / 100;
    return subtotal + tax;
}

// file1.cpp
#include "math_helper.h"
int calculateTotal(int price, int quantity) {
    return calculateTotalWithTax(price, quantity);
}

// file2.cpp
#include "math_helper.h"
int computeAmount(int cost, int count) {
    return calculateTotalWithTax(cost, count);
}
```

Boy-scout detects three types of duplication:
- **Type-1 (exact):** Identical token sequences
- **Type-2 (renamed):** Same structure with different identifier names
- **Type-3 (near-miss):** Structurally similar but not identical (LCS similarity ≥ 0.70)
