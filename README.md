# **ShX – A Lightweight POSIX Shell in Go**

ShX is a minimalistic, **POSIX-compliant** command-line shell written in **Go**. It interprets and executes both built-in commands and external programs, providing a functional and efficient shell experience.

This project is a great learning resource for **command parsing, REPL (Read-Eval-Print Loop), process management, and shell scripting**.

---

## **Features**

✅ **Built-in Commands**  
- Supports essential commands like `cd`, `pwd`, `echo`, `exit`, and `type`.

✅ **External Command Execution**  
- Runs system executables found in `$PATH`.

✅ **Command Parsing**  
- Handles single & double quotes, escaping (`\`), and arguments.

✅ **Redirection Operators**  
- `>` / `1>` : Overwrite `stdout`.  
- `>>` / `1>>` : Append to `stdout`.  
- `2>` : Overwrite `stderr`.  
- `2>>` : Append to `stderr`.

✅ **Autocompletion**  
- **Single Tab**: Auto-completes when there’s a unique match or common prefix.  
- **Double Tab**: Lists all possible matches if multiple exist.

✅ **Customized Prompt**  
- Displays a **green** `ShX` prompt with an arrow (`➜`) before input:  
  ```go
  fmt.Fprint(os.Stdout, "\r\033[1;32mShX\033[0m ➜ ")
  ```

---

## **Installation**  

### **Prerequisites**  
Ensure **Go** is installed. [Install Go](https://go.dev/doc/install).  

### **Clone & Build**  
```bash
git clone https://github.com/ShreyamKundu/ShX.git
cd ShX/cmd/
go build -o ShX main.go
```

---

## **Usage**

Start ShX by running:

```bash
./ShX
```

You'll see a prompt:

```bash
ShX ➜ 
```

Now you can enter commands.

---

## **1. `echo` – Print Messages**

### **Basic Usage**
```bash
ShX ➜ echo Hello, ShX!
Hello, ShX!
```

### **With Quotes**
```bash
ShX ➜ echo 'Hello, ShX!'
Hello, ShX!
```

### **With Escape Characters**
```bash
ShX ➜ echo Hello\,\ ShX\!
Hello, ShX!
```

---

## **2. `pwd` – Print Current Directory**
```bash
ShX ➜ pwd
/home/user/shx
```

**Redirect Output to File:**
```bash
ShX ➜ pwd > cwd.txt
```

---

## **3. `cd` – Change Directory**

### **Change to an Absolute Path**
```bash
ShX ➜ cd /usr/local
```

### **Relative Path Navigation**
```bash
ShX ➜ cd ./bin
```

### **Move Up a Directory (`..`)**
```bash
ShX ➜ cd ..
```

### **Go to Home Directory (`~`)**
```bash
ShX ➜ cd ~
```

---

## **4. `type` – Identify Commands**

### **Check a Built-in Command**
```bash
ShX ➜ type cd
cd is a shell builtin
```

### **Check an External Command**
```bash
ShX ➜ type ls
ls is /usr/bin/ls
```

### **Check an Unknown Command**
```bash
ShX ➜ type unknowncmd
unknowncmd: not found
```

---

## **5. `exit` – Quit ShX**
```bash
ShX ➜ exit
```
✅ Ends the session.

---

## **6. Redirection Operators**

### **Standard Output (`>` and `>>`)**
**Overwrite Output:**
```bash
ShX ➜ echo "Hello, ShX!" > output.txt
```
**Append Output:**
```bash
ShX ➜ echo "Appending this line" >> output.txt
```

### **Standard Error (`2>` and `2>>`)**
**Overwrite Errors:**
```bash
ShX ➜ ls /invalidpath 2> error.log
```
**Append Errors:**
```bash
ShX ➜ ls /invalidpath 2>> error.log
```

---

## **7. Autocompletion in ShX**

### **Single Tab Completion**
If there’s only **one** match:
```bash
ShX ➜ pw<Tab>
ShX ➜ pwd
```

### **Double Tab Completion**
If multiple matches exist:
```bash
ShX ➜ e<Tab><Tab>
echo  exit
```

### **External Command Autocompletion**
```bash
ShX ➜ bas<Tab>
ShX ➜ bash
```

---

## **8. External Command Execution**
ShX executes external commands like `ls`, `grep`, and `vim` if found in `$PATH`:

```bash
ShX ➜ ls
main.go  shx  README.md
```

**Handling Unknown Commands:**
```bash
ShX ➜ unknowncmd
unknowncmd: command not found
```

---

## **Future Enhancements**
### ✅ **Piping (`|`)**
- Example: `ls | grep main`
- Allows chaining commands.

### ✅ **Command History**
- Implementing `history` to recall past commands.
- Storing history in `~/.shx_history`.

### ✅ **Job Control**
- Support for **background (`bg`)** and **foreground (`fg`)** jobs.
- Implementing `jobs` to list running jobs.
- Handling **CTRL+Z** to stop processes.

---

ShX is a **lightweight yet powerful shell**, ideal for learning and customization. Contributions and feature suggestions are always welcome!

Enjoy using **ShX** and happy coding! 🚀
