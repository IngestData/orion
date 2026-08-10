# Orion - hunter of the hidden

Orion is a SCA tool deployed within IngestData Pipe. 

It analyzes transformer code for potential secuirity vulnerabilities and determines if the code is safe to run.
The tool runs after each change of a projects transformer code to ensure that the code is safe to run and does not contain any vulnerabilities.  

If vulnerabilities are found, Orion will report them to the user and prevent the code from being deployed until the vulnerabilities are fixed.   

In addition to Vulnerabilites, Memory leaks and other issues Orion will also check for any kind of network requests that the transformer code might be making. This is to ensure that the code is not making any unauthorized network requests that could compromise the security of the system.  

Beyond scanning the transformer code in order to determine its intentions, orion must also ensure that the transformer is always sanatizing its input and output. This is to ensure that the transformer code is not passing any malicious content through pipe as code and later executing it in the transformer runtime.

