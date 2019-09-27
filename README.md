# ting-bill-split
Split your Ting bills based on usage for each line.

## Summary
Ting provides cellular service in the United States using Sprint and T-Mobile networks - [Wikipedia](https://en.wikipedia.org/wiki/Ting_Inc.)

Many families will put all their lines on the same plan to share savings, but Ting's billing system doesn't provide guidance on how to proportionately split monthly bills based on each line's usage. The basic solution is to split the monthly total cost evenly by number of lines. Unfortunately, this way, one line streaming videos all the time would negate the savings of frugal data usage for other people on the plan.

With `ting-bill-split`, the plan owner can use the Ting-provided `*.csv` each month to split the bill more intelligently. Shared costs are split _**evenly**_ between all lines, and the usage-based costs are split _**proportionately**_.

## Installing
In the future, I might supply pre-compiled binaries so that you don't need to install Go.

### First Installation
1. [Install Go for your operating system](https://golang.org/doc/install)
1. In a terminal, type 
   ```
   go get github.com/hitjim/ting-bill-split
   ```

### Installing Updates
In a terminal, type
```
go get -u github.com/hitjim/ting-bill-split
```

## Usage

### Getting Help
For help, type into your terminal
```
ting-bill-split help
```
This will print a usage guide for working with a directory containing the required files for splitting a single month's bill.

### Working With Bill Directories
The primary method for splitting bills is to work with directories. Here are the main paths for creating and using a bill directory

