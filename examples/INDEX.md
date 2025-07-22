# Workie Examples Index

This directory contains comprehensive examples and templates for using workie across different project types and development scenarios.

## 📁 Project Templates

### Core Examples

| File | Description | Best For |
|------|-------------|----------|
| [`minimal.workie.yaml`](minimal.workie.yaml) | Simplest possible configuration | Quick starts, learning workie basics |
| [`web-app.workie.yaml`](web-app.workie.yaml) | Full-stack web application | React + Node.js projects |
| [`django-project.workie.yaml`](django-project.workie.yaml) | Python Django REST API | Django web applications |
| [`go-microservice.workie.yaml`](go-microservice.workie.yaml) | Go microservice with Docker | Go APIs and services |
| [`mobile-app.workie.yaml`](mobile-app.workie.yaml) | React Native mobile app | iOS/Android development |
| [`data-science.workie.yaml`](data-science.workie.yaml) | ML/Data analysis project | Jupyter, MLflow, data pipelines |
| [`rust-project.workie.yaml`](rust-project.workie.yaml) | Rust application | Systems programming, CLI tools |
| [`docker-compose.workie.yaml`](docker-compose.workie.yaml) | Multi-service Docker setup | Containerized environments |

## 🔄 Workflow Examples

### Specialized Workflows

| File | Description | Use Case |
|------|-------------|----------|
| [`workflows/ci-cd.workie.yaml`](workflows/ci-cd.workie.yaml) | CI/CD pipeline templates | GitHub Actions, GitLab CI |
| [`workflows/feature-development.workie.yaml`](workflows/feature-development.workie.yaml) | Feature branch workflow | Git flow, pull requests |

## 🚀 Quick Start Guide

### 1. Choose Your Template

Browse the examples above and find the one that best matches your project type.

### 2. Copy and Customize

```bash
# Copy the template to your project
cp examples/web-app.workie.yaml .workie.yaml

# Edit for your specific needs
vim .workie.yaml
```

### 3. Get Started

```bash
# Install dependencies
workie run install

# Start development
workie run dev
```

## 📊 Feature Comparison

| Feature | Minimal | Web App | Django | Go | Mobile | Data Science | Rust | Docker |
|---------|---------|---------|--------|----|----|--------------|------|--------|
| **Basic Tasks** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Database Setup** | ❌ | ✅ | ✅ | ✅ | ❌ | ✅ | ✅ | ✅ |
| **File Watching** | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Git Hooks** | ❌ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ❌ |
| **Testing** | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ | ✅ |
| **Docker Integration** | ❌ | ❌ | ❌ | ✅ | ❌ | ❌ | ❌ | ✅ |
| **Platform Support** | Any | Web | Web | Any | Mobile | Any | Any | Any |

## 🎯 Use Case Recommendations

### Starting a New Project?
- **Web Application**: [`web-app.workie.yaml`](web-app.workie.yaml)
- **API Service**: [`go-microservice.workie.yaml`](go-microservice.workie.yaml) or [`django-project.workie.yaml`](django-project.workie.yaml)
- **Mobile App**: [`mobile-app.workie.yaml`](mobile-app.workie.yaml)
- **Data Analysis**: [`data-science.workie.yaml`](data-science.workie.yaml)
- **Simple Project**: [`minimal.workie.yaml`](minimal.workie.yaml)

### Converting Existing Project?
1. Start with [`minimal.workie.yaml`](minimal.workie.yaml)
2. Add features gradually from other examples
3. Test each addition before moving to the next

### Team Development?
- Use [`workflows/feature-development.workie.yaml`](workflows/feature-development.workie.yaml)
- Integrate with [`workflows/ci-cd.workie.yaml`](workflows/ci-cd.workie.yaml)

## 🛠 Customization Patterns

### Common Modifications

1. **Environment Variables**
   ```yaml
   env:
     NODE_ENV: development
     API_KEY: ${API_KEY}
   ```

2. **Task Dependencies**
   ```yaml
   tasks:
     dev:
       depends_on: ["install", "db:setup"]
   ```

3. **Custom Scripts**
   ```yaml
   scripts:
     reset:
       description: "Reset everything"
       commands:
         - "rm -rf node_modules"
         - "workie run install"
   ```

4. **File Watching**
   ```yaml
   watch:
     - pattern: "**/*.js"
       tasks: ["lint", "test"]
   ```

## 📝 Contributing Examples

Have a useful configuration? Help others by contributing:

1. Create a new example file
2. Follow the naming convention: `{project-type}.workie.yaml`
3. Include comprehensive documentation
4. Test thoroughly
5. Submit a pull request

## 🆘 Getting Help

- **Documentation**: Check the main [README](../README.md)
- **Similar Examples**: Browse this index for related use cases
- **Community**: File issues on the project repository

## 📚 Additional Resources

- [Main Documentation](../README.md)
- [Configuration Reference](../docs/configuration.md)
- [Best Practices](../docs/best-practices.md)

---

**Legend:**
- ✅ Included and configured
- ❌ Not included
- 📁 Directory
- 🔄 Workflow
- 🚀 Quick start
- 📊 Comparison
- 🎯 Recommendations
- 🛠 Customization
- 📝 Contributing
- 🆘 Support
- 📚 Resources
