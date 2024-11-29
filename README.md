# OpenAuth
Configurable User Authentication Server in K8s

# Build
target: `build-server`, `build-cli`
the program will be located in `build/`
```bash
make all
```
miscelenous
```bash
make clean     
make test     
make deps     
make fmt      
make lint 
```

# Directory Structure
- `/cmd`: main application program
- `/pkg`: library codes that is okay to imported from external project and application
- `/api`: OpenAPI/Swagger Spec, JSON schemas, Protocol Definition Files
- `/configs`: Configuration File or Templates
- `/scripts`: directory including for building, installing, pushing dockerfile on registry scripts
- `/build`: Packaging and CI/CD
- `/deployments`: The program for requesting some of the codes
- `/docs`: design and architecture description files
- `/example`: external application or opened library examples
- `/assets`: repository images