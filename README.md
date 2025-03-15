# GavelEngine

**GavelEngine** is a flexible, JSON-based rules engine library for Go. It empowers you to define dynamic facts and rules with nested conditions, rule chaining, and custom operator decorators. Just like a judge’s gavel brings finality to a case, GavelEngine evaluates complex business logic and delivers decisive outcomes in your Go applications.

---

## Features

- **Dynamic Fact Evaluation**  
  Define facts as constant values or functions that compute values at runtime.

- **Robust Rule Evaluation**  
  Create rules with nested conditions using `all`, `any`, and `not` constructs for precise decision-making.

- **Custom Operators & Decorators**  
  Easily extend built-in operators with decorators (using a colon syntax) for features such as case-insensitive comparisons and operand swapping.

- **Rule Chaining**  
  Enable rules to trigger additional evaluations by dynamically setting runtime facts during engine execution.

- **Simplicity & Extensibility**  
  A clean, modular design that lets you get started quickly while remaining flexible for complex scenarios.

---

## Installation

Install GavelEngine with `go get`:

```bash
go get github.com/yourusername/GavelEngine

