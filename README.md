# About
Brings mouse, keyboard and motion control to CEMU with use of DSU controller having default mapping for BOTW


# Building and usage
Install golang package for operating system of your choice - Windows, Linux or MacOS. Build repo with "go build ." command. Run resulting binary within terminal. Give accessibility permissions once asked (for sure you will be prompted on MacOS). These permissions are required to track keystrokes and mouse movements in background. Launch CEMU and connect to DSU Controller (served by this running process). Rest of features and steps are in development so stay tuned.