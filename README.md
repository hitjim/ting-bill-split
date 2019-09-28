# ting-bill-split
Split your Ting bills based on usage for each line.

## Summary
Ting provides cellular service in the United States using Sprint and T-Mobile networks - [Wikipedia](https://en.wikipedia.org/wiki/Ting_Inc.)

Many families will put all their lines on the same plan to share savings, but Ting's billing system doesn't provide guidance on how to proportionately split monthly bills based on each line's usage. The basic solution is to split the monthly total cost evenly by number of lines. Unfortunately, this way, one line streaming videos all the time would negate the savings of frugal data usage for other people on the plan.

With `ting-bill-split`, the plan owner can use the Ting-provided `*.csv` each month to split the bill more intelligently. Shared costs are split _**evenly**_ between all lines, and the usage-based costs are split _**proportionately**_.

## Requirements
This guide assumes you can:
* View the "Monthly bill" `.pdf` for a given billing month from [Ting's "Billing history" page](https://ting.com/account/bill_history)
* You can download `.csv` files for `minutes`, `messages`, `megabytes`, available by clicking on the "Monthly bill" link for the month, then the download arrow under the "CSV" column.
* You have sufficient permissions on your computer required to [install and run Go programs](https://golang.org/doc/install) (you probably do!)

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
The primary method for splitting bills is to work with directories in "batch mode".

1. In your terminal, navigate to the directory in which you wish to create and save your bill data. For example:
   ```
   cd ~/ting-bills
   ```
1. Create a new directory, preferably provide a name. If you don't provide a name, the new directory will be assigned by default. Here's an example with a date:
   ```
   ting-bill-split new 2019-09-ting
   ```
   This will create a new directory with the name provided. Inside will be a `bill.toml` file where you can fill in the required information about your monthly bill.
1. Update the info in `bill.toml` to reflect the respective info for plan and that months billing.
   * **_NOTE_** - If you have a previous month's `bill.toml`, you can usually use it to replace the new one, and update
1. Download and move all the `.csv` files for the month into this directory.
1. Run `ting-bill-split`, here are some examples:
   * From the same directory as when `ting-bill-split new <dir>` was run, if `<dir>` is `2019-09-ting`:
      ```
      ting-bill-split dir 2019-09-ting
      ```
   * From anywhere:
      ```
      ting-bill-split dir <full path to dir>
      ```
      for example...
      ```
      ting-bill-split dir ~/2019-09-ting
      ```
      ```
      ting-bill-split dir /home/yourusername/ting-bills/2019-09-ting
      ```
   * From inside the directory containing your `.csv` files and `bill.toml`
      ```
      ting-bill-split dir .
      ```
1. Review the resulting `.pdf` file in the bill split directory you chose.
1. For each following month's bill, you can either:
   * Start again at **_step #2_**
   * Make a new directory manually, copy the previous month's `bill.toml` into it, start at **_step #3_**

## Breakdown of `bill.toml` Info
* **`description`** - Ideally this is a unique string of characters, I recommend including the billing date. This description is used as part of the resulting `.pdf` file after calculating the bill split.
* **`deviceIds`** - Each string is a unique phone number on the Ting plan.
   * **_NOTE_**: do NOT use dashes. _Example_: `"1112223333"`, not `"111-222-3333"`.
* `shortStrawId` - In the unlikely event a cost can't be split evenly between lines, this is the line that will absorb that cost. It's usually $0.01, and I usually use the plan owner's number (probably you!). This is due to math, our inability to split pennies in half, and partially a personal judgement call based on complexity and ROI :)
* The rest of the values are US Dollar amounts and use a decimal format to suit. _Example:_ `48.00`, not `48` or `"48.00"`.
   * **`total`** - This is the final cost of the month's bill.
   * **`devices`** - This is the shared cost based on how many lines or devices are on the plan, and is provided in the Ting bill.
   * **`minutes`, `messages`, `megabytes`, `extraMinutes` etc...** - These reflect the usage cost breakdowns, and are provided in the Ting bill for each type.
   * **`fees`** - This is the total of all the "Taxes and regulatory fees" Ting is required to collect, and is provided in the Ting bill. This is a shared cost, and `ting-bill-split` doesn't use the individual costs.

## Extra Program Usage Info
* You can rename the `.csv` files you get from Ting. As long as "messages", "minutes", and "megabytes" is part of the filename for the respective files, "batch mode" will still work.
* You can move the lines in the `bill.toml` file, perhaps grouping in a way you prefer. But each line is required in the format provided in the original file.
* Include **_every number_** listed by Ting for that month's charges. Do so even if a line is suspended for the entire month, or deactivated for part of it. This line will still incur charges despite reduced or zero usage, and thus affects how the shared costs are split per line.

