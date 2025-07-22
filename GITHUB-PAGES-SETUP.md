# GitHub Pages Setup for Workie Documentation

This guide will help you set up GitHub Pages to host your Workie documentation online.

## Prerequisites

- Your repository must be public (or you need GitHub Pro/Team for private repos)
- You must have admin access to the repository

## Setup Steps

### 1. Enable GitHub Pages

1. Go to your repository on GitHub: `https://github.com/[username]/workie`
2. Click on **Settings** tab
3. Scroll down to **Pages** in the left sidebar
4. Under **Source**, select **Deploy from a branch**
5. Choose **main** branch and **/ (root)** folder
6. Click **Save**

### 2. Alternative: Use docs/ folder

If you prefer to serve documentation from the `/docs` folder:

1. Follow steps 1-4 above
2. Choose **main** branch and **/docs** folder
3. Click **Save**

### 3. Custom Domain (Optional)

If you have a custom domain:

1. In the **Pages** settings, enter your custom domain
2. Enable **Enforce HTTPS**
3. Add a CNAME record in your DNS pointing to `[username].github.io`

## What Gets Published

With the current setup, GitHub Pages will serve:

- **README.md** as the homepage (automatically converted to HTML)
- All documentation in **docs/** folder
- **USAGE.md** for usage instructions
- **CONTRIBUTING.md** for contribution guidelines
- **CODE_OF_CONDUCT.md** for community standards

## Accessing Your Documentation

Once enabled, your documentation will be available at:

- **GitHub Pages URL**: `https://[username].github.io/workie/`
- **Custom Domain** (if configured): `https://yourdomain.com/`

## Documentation Structure

The current documentation structure includes:

```
docs/
├── api/                 # API documentation
├── architecture/        # System architecture docs
├── deployment/          # Deployment guides
├── examples/            # Usage examples
└── troubleshooting/     # Common issues and solutions
```

## Customizing the Site

### Using Jekyll (Recommended)

GitHub Pages supports Jekyll out of the box. To customize:

1. Add a `_config.yml` file in your repository root:

```yaml
title: Workie
description: A flexible worker queue system for distributed task processing
theme: minima
plugins:
  - jekyll-feed
  - jekyll-sitemap

markdown: kramdown
highlighter: rouge

# Navigation
header_pages:
  - README.md
  - USAGE.md
  - docs/api/README.md
  - CONTRIBUTING.md

# SEO
url: "https://[username].github.io"
baseurl: "/workie"
```

2. Create a `_layouts` folder for custom page templates
3. Add a `_includes` folder for reusable components

### Custom CSS

Add custom styling by creating:

```
assets/
└── css/
    └── style.scss
```

With content:
```scss
---
---

@import "minima";

// Your custom styles here
.highlight {
    background-color: #f8f8f8;
}
```

## Monitoring and Analytics

Consider adding:

1. **Google Analytics** - Add tracking ID to `_config.yml`
2. **GitHub repository metrics** - Display stars/forks
3. **Documentation search** - Using Algolia or similar

## Best Practices

1. **Keep README.md comprehensive** - It becomes your homepage
2. **Use relative links** - Ensures links work on GitHub and Pages
3. **Add navigation** - Help users find relevant documentation
4. **Include examples** - Code samples with syntax highlighting
5. **Mobile-friendly** - Ensure responsive design

## Troubleshooting

### Common Issues

1. **404 errors**: Check that files exist and paths are correct
2. **Build failures**: Check the Pages tab for build logs
3. **CSS not loading**: Verify asset paths and Jekyll configuration
4. **Links broken**: Use relative paths where possible

### Getting Help

- [GitHub Pages Documentation](https://docs.github.com/en/pages)
- [Jekyll Documentation](https://jekyllrb.com/docs/)
- [GitHub Community Forum](https://github.community/)

## Updating Documentation

To update your live documentation:

1. Make changes to your documentation files
2. Commit and push to the main branch
3. GitHub Pages will automatically rebuild (usually within a few minutes)

Your documentation site will be automatically updated whenever you push changes to the main branch!
