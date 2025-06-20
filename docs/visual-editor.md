# Visual Workflow Editor

The Operion Visual Editor is a React-based web interface that provides an intuitive, node-based visual representation of workflows stored in the Operion system.

## Overview

The visual editor transforms JSON-defined workflows into interactive, visual flow diagrams using a modern web stack. It serves as the primary user interface for viewing and understanding workflow structures, with planned capabilities for editing and creating workflows.

## Technology Stack

### Core Framework
- **React 19.1.0** with **TypeScript** for type-safe component development
- **Vite 6.3.5** for fast development and optimized builds
- **React Router 7.6.2** for single-page application navigation

### Visual Flow Engine
- **@xyflow/react 12.7.0** (ReactFlow) - Powers the node-based visual editor
- **@dagrejs/dagre 1.1.5** - Automatic graph layout and positioning algorithms

### UI Design System
- **Tailwind CSS 4.1.10** - Utility-first styling with custom design tokens
- **Radix UI Components** - Accessible, unstyled component primitives
- **Lucide React** - Consistent icon system
- **Inter** and **Source Code Pro** fonts for optimal readability

### Additional Libraries
- **date-fns** - Date formatting and manipulation
- **Custom API Client** - Typed HTTP client for Operion REST API

## Architecture

### Directory Structure
```
ui/operion-editor/
├── src/
│   ├── components/
│   │   ├── flow/           # ReactFlow node and edge components
│   │   ├── ui/             # Reusable design system components
│   │   ├── workflow/       # Workflow-specific display components
│   │   ├── devtools/       # Development debugging tools
│   │   └── layout/         # Page layout and navigation
│   ├── pages/
│   │   ├── Home.tsx        # Workflow dashboard and listing
│   │   └── workflows/Get.tsx # Individual workflow visualization
│   ├── types/operion.ts    # TypeScript type definitions
│   ├── lib/api-client.ts   # API communication layer
│   └── styles/             # Global styles and Tailwind config
```

### Component Architecture
- **Page Components** - Route-level components for major application views
- **Layout Components** - Reusable layout patterns and containers
- **Flow Components** - ReactFlow-specific nodes, edges, and controls
- **UI Components** - Design system primitives (buttons, cards, badges)
- **Workflow Components** - Domain-specific workflow visualization logic

## Features

### Workflow Dashboard
- **Workflow Grid** - Card-based display of all workflows with metadata
- **Status Indicators** - Visual badges showing workflow states (active/inactive/error)
- **Metadata Display** - Owner, last updated, step count, and description
- **Quick Navigation** - Direct links to open workflows in the visual editor
- **Search and Filtering** - Find workflows by name, status, or metadata

### Visual Flow Editor
- **Node-Based Visualization** - Each workflow step represented as a connected node
- **Automatic Layout** - Intelligent positioning using Dagre graph algorithms
- **Node Type Differentiation** - Visual distinction between triggers and action steps
- **Connection Visualization** - Success and failure paths shown as colored edges

### Node Types and Visual Design

#### Trigger Nodes
- **Appearance** - Green-themed with satellite dish icon
- **Purpose** - Represent workflow initiation points (schedule, webhook, etc.)
- **Positioning** - Always positioned as entry points to the workflow

#### Action Step Nodes
Action nodes are color-coded by type:
- **HTTP Request** - Blue with cloud icon for API calls
- **Transform** - Orange with function icon for data processing
- **File Write** - Yellow with pen icon for file operations
- **Default** - Gray with box icon for unrecognized action types

### Connection Types
- **Success Connections** - Green edges showing normal execution flow
- **Failure Connections** - Red edges showing error handling paths
- **Animated Edges** - Visual feedback with directional arrow markers

### Development Tools
- **Node Inspector** - Real-time inspection of node data and properties
- **Change Logger** - Debug panel tracking flow modifications
- **Viewport Logger** - Monitor zoom levels and viewport changes
- **DevTools Panel** - Comprehensive debugging interface for development

## API Integration

### REST API Client
Located in `src/lib/api-client.ts`, provides:
- **Workflow Fetching** - Retrieve individual workflows and workflow lists
- **Error Handling** - RFC7807 problem format support
- **Type Safety** - Full TypeScript integration with Operion domain models
- **Base URL Configuration** - Connects to Operion API server at `http://localhost:3000`

### Data Flow
1. **API Requests** - Fetch workflow data from Operion backend
2. **Type Transformation** - Convert JSON to TypeScript domain models
3. **Flow Conversion** - Transform workflow steps into ReactFlow nodes and edges
4. **Layout Calculation** - Use Dagre algorithm for automatic positioning
5. **Rendering** - Display interactive visual representation

## Development

### Getting Started
```bash
# Navigate to the editor directory
cd ui/operion-editor

# Install dependencies
npm install

# Start development server
npm run dev
```

The development server runs on `http://localhost:5173` with hot module replacement.

### Build Process
```bash
# Production build
npm run build

# Preview production build
npm run preview

# Lint TypeScript and React code
npm run lint
```

### Configuration
- **Vite Config** - Modern build tooling with TypeScript support
- **Tailwind Config** - Custom design system tokens and utilities
- **TypeScript Config** - Strict type checking with path aliases
- **ESLint Config** - Code quality rules for React and TypeScript

## Current Limitations

### Read-Only Interface
The current implementation is primarily focused on visualization:
- **No Editing** - Cannot modify existing workflows through the UI
- **No Creation** - Cannot create new workflows via the interface
- **No Deletion** - Cannot remove workflows through the visual editor

### Planned Enhancements
- **Drag-and-Drop Editing** - Visual workflow construction
- **Node Property Editing** - Inline editing of action configurations
- **Workflow Creation** - Complete workflow authoring capabilities
- **Registry Integration** - Dynamic loading of available actions and triggers
- **Real-time Updates** - Live workflow status and execution monitoring

## Integration with Operion System

### Workflow Data Source
- Reads workflows from Operion's file-based persistence (`./data/workflows/`)
- Connects to Operion API server for data retrieval
- Supports all workflow features defined in the Operion domain models

### Future Integration Points
- **Registry System** - Dynamic discovery of available actions and triggers
- **Execution Monitoring** - Real-time workflow execution status
- **Event System** - Live updates via WebSocket or Server-Sent Events
- **Authentication** - User management and access control

## Browser Compatibility

The visual editor targets modern browsers with:
- **ES2020** JavaScript features
- **CSS Grid** and **Flexbox** layout
- **WebAssembly** support (for advanced computations)
- **Modern React** features (concurrent rendering, automatic batching)

## Performance Considerations

- **Code Splitting** - Dynamic imports for reduced initial bundle size
- **Virtual Scrolling** - Efficient rendering of large workflow lists
- **Memoization** - React.memo for expensive component renders
- **Lazy Loading** - On-demand loading of workflow details