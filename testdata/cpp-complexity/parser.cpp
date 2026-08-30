// parser.cpp - test fixture for C++ complexity analysis
#include <string>
#include <vector>

// This function has cyclomatic complexity of 7 (base 1 + 6 if statements)
// It should be flagged when max-complexity is set to 6
std::string parseStatement(const std::string& input) {
  if (input[0] == 'i') {
    if (input.length() > 1) {
      if (input[1] == 'f') {
        if (input.length() > 2) {
          if (input[2] == ' ') {
            if (input.find("then") != std::string::npos) {
              return "if-then-statement";
            }
          }
        }
      }
    }
  }
  return "statement";
}

// This function has complexity 2 (base 1 + 1 for loop)
// It should not be flagged at default limit of 6
void simpleLoop(int n) {
  for (int i = 0; i < n; ++i) {
    doSomething();
  }
}

void doSomething() {
  // stub
}
