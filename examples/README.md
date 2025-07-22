# Workie Examples

This directory contains sample `.workie.yaml` configurations for different project types and development scenarios. These examples help you get started quickly with workie by providing proven patterns and configurations.

## Quick Start

1. Browse the examples below to find one that matches your project type
2. Copy the relevant `.workie.yaml` file to your project root
3. Customize the configuration for your specific needs
4. Run `workie --help` to see available commands

## Available Examples

### Web Development

- **[web-app.workie.yaml](web-app.workie.yaml)** - Full-stack web application with React frontend and Node.js backend
  - Features: Database setup, development servers, testing, linting
  - Includes: PostgreSQL, Redis, npm scripts integration

- **[minimal.workie.yaml](minimal.workie.yaml)** - Minimal configuration for any project
  - Perfect for: Getting started quickly, simple projects
  - Includes: Basic install, dev, test, and build tasks

### Backend Development

- **[django-project.workie.yaml](django-project.workie.yaml)** - Django REST API project
  - Features: Virtual environment management, database migrations, Django admin
  - Includes: PostgreSQL, Redis, code formatting, test coverage

- **[go-microservice.workie.yaml](go-microservice.workie.yaml)** - Go microservice with Docker
  - Features: Hot reloading, testing, benchmarks, Docker integration
  - Includes: PostgreSQL, Redis, code generation, API documentation

### Mobile Development

- **[mobile-app.workie.yaml](mobile-app.workie.yaml)** - React Native mobile application
  - Features: iOS and Android builds, device deployment, debugging
  - Includes: Metro bundler, platform-specific configurations, release builds

### Data Science

- **[data-science.workie.yaml](data-science.workie.yaml)** - Machine learning and data analysis project
  - Features: Jupyter integration, MLflow tracking, data pipelines
  - Includes: Notebook management, model training, experiment tracking

## Common Configuration Patterns

### Task Dependencies

```yaml
tasks:
  build:
    description: "Build the application"
    commands:
      - "npm run build"
    depends_on: ["install", "test"]
```

### Environment Variables

```yaml
env:
  NODE_ENV: development
  DATABASE_URL: postgresql://localhost:5432/myapp_dev
  API_KEY: ${API_KEY}  # Use environment variables from shell
```

### File Watching

```yaml
watch:
  - pattern: "**/*.js"
    tasks: ["lint", "test"]
  - pattern: "**/*.css"
    tasks: ["build:css"]
```

### Git Hooks Integration

```yaml
hooks:
  pre-commit:
    - "workie run lint"
    - "workie run test"
  pre-push:
    - "workie run build"
```

### Custom Scripts

```yaml
scripts:
  db:reset:
    description: "Reset database completely"
    commands:
      - "dropdb myapp_dev --if-exists"
      - "createdb myapp_dev"
      - "workie run migrate"
```

## Workflow Examples

### Development Workflow

```bash
# Initial setup
workie run install
workie run db:setup

# Daily development
workie run dev        # Start development servers
workie run test       # Run tests
workie run lint       # Check code quality
```

### CI/CD Integration

```bash
# In your CI pipeline
workie run install
workie run lint
workie run test
workie run build
```

### Team Onboarding

```bash
# New team member setup
git clone <repository>
cd <project>
workie run setup      # One command to rule them all
```

## Customization Tips

### 1. Start Simple
Begin with the minimal example and gradually add features as needed.

### 2. Use Task Dependencies
Leverage `depends_on` to ensure prerequisites are met:

```yaml
tasks:
  test:
    depends_on: ["install", "db:migrate"]
```

### 3. Environment-Specific Configurations

```yaml
environments:
  development:
    env:
      DEBUG: "true"
  production:
    env:
      DEBUG: "false"
```

### 4. Document Your Setup

Include comprehensive documentation in your configuration:

```yaml
docs:
  setup: |
    ## Getting Started
    1. Install dependencies: `workie run install`
    2. Start development: `workie run dev`
```

## Best Practices

1. **Keep it DRY**: Use task dependencies and scripts to avoid repetition
2. **Document everything**: Include clear descriptions for all tasks
3. **Use meaningful names**: Task names should be intuitive and consistent
4. **Test your configuration**: Regularly verify that all tasks work as expected
5. **Version control**: Include your `.workie.yaml` in version control

## Contributing

Have a useful configuration pattern or example? Consider contributing it back to the workie project:

1. Create a new example file following the naming convention
2. Include comprehensive documentation and comments
3. Test the configuration thoroughly
4. Submit a pull request

## Need Help?

- Check the [main documentation](../README.md)
- Look at existing examples for similar use cases
- File an issue on the project repository for specific questions

## Example Usage

### Starting a New Web Project

```bash
# Copy the web app template
cp examples/web-app.workie.yaml .workie.yaml

# Customize for your project
vim .workie.yaml

# Run the setup
workie run install
workie run dev
```

### Converting an Existing Project

1. Start with the minimal template
2. Add tasks one by one based on your current workflow
3. Test each task as you add it
4. Gradually enhance with dependencies, watching, and hooks

Happy coding with workie! 🚀
