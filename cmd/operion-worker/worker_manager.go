package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dukex/operion/pkg/eventbus"
	"github.com/dukex/operion/pkg/events"
	"github.com/dukex/operion/pkg/models"
	"github.com/dukex/operion/pkg/persistence"
	"github.com/dukex/operion/pkg/registry"
)

type WorkerManager struct {
	id               string
	logger           *slog.Logger
	persistence      persistence.Persistence
	registry         *registry.Registry
	eventBus         eventbus.EventBus
	inputCoordinator *InputCoordinator
}

func NewWorkerManager(
	id string,
	persistence persistence.Persistence,
	eventBus eventbus.EventBus,
	logger *slog.Logger,
	registry *registry.Registry,
) *WorkerManager {
	return &WorkerManager{
		id:               id,
		logger:           logger.With("module", "operion-worker", "worker_id", id),
		persistence:      persistence,
		registry:         registry,
		eventBus:         eventBus,
		inputCoordinator: NewInputCoordinator(persistence, logger),
	}
}

func (w *WorkerManager) Start(ctx context.Context) error {
	w.logger.InfoContext(ctx, "Starting worker manager with node-based architecture", "worker_id", w.id)

	// Updated for node-based architecture: handle node activations instead of step availability
	err := w.eventBus.Handle(events.NodeActivationEvent, w.handleNodeActivation)
	if err != nil {
		return err
	}

	err = w.eventBus.Subscribe(ctx)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to subscribe to event bus", "error", err)

		return err
	}

	w.logger.InfoContext(ctx, "Worker started successfully with node-based execution")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	w.logger.InfoContext(ctx, "Shutting down worker...")

	return nil
}

// handleNodeActivation handles node activation events - this is the core of the new node-based execution
// Each worker can execute nodes independently and coordinate through direct node-to-node activation.
func (w *WorkerManager) handleNodeActivation(ctx context.Context, event any) error {
	nodeActivationEvent, ok := event.(*events.NodeActivation)
	if !ok {
		w.logger.ErrorContext(ctx, "Invalid event type for NodeActivation")

		return nil
	}

	logger := w.logger.With(
		"published_workflow_id", nodeActivationEvent.PublishedWorkflowID,
		"execution_id", nodeActivationEvent.ExecutionID,
		"node_id", nodeActivationEvent.NodeID,
	)

	logger.InfoContext(ctx, "Processing node activation event")

	// 1. Get node definition from NodeRepository
	node, err := w.persistence.NodeRepository().GetNodeFromPublishedWorkflow(ctx, nodeActivationEvent.PublishedWorkflowID, nodeActivationEvent.NodeID)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to get node definition", "error", err)

		return w.publishNodeCompletionEvent(ctx, nodeActivationEvent, nil, err)
	}

	if node == nil {
		err = fmt.Errorf("node not found: %s", nodeActivationEvent.NodeID)
		logger.ErrorContext(ctx, "Node not found", "error", err)

		return w.publishNodeCompletionEvent(ctx, nodeActivationEvent, nil, err)
	}

	// 2. Get node input requirements
	requirements := w.getNodeInputRequirements(ctx, node)

	// 3. Check if node has a pending execution (FIFO for loops)
	pendingExecution, err := w.inputCoordinator.GetPendingNodeExecution(
		ctx,
		nodeActivationEvent.NodeID,
		nodeActivationEvent.ExecutionID,
	)
	if err != nil {
		logger.WarnContext(ctx, "Failed to check pending executions", "error", err)
		// Continue with new execution if we can't check pending ones
	}

	var nodeExecutionID string
	if pendingExecution != nil {
		// Use existing pending execution (FIFO)
		nodeExecutionID = pendingExecution.NodeExecutionID
		logger.DebugContext(ctx, "Using existing pending node execution", "node_execution_id", nodeExecutionID)
	} else {
		// Create new node execution instance
		nodeExecutionID = GenerateNodeExecutionID()
		logger.DebugContext(ctx, "Created new node execution", "node_execution_id", nodeExecutionID)
	}

	// 4. Add this input to coordination state
	// Convert input data to map[string]any format expected by NodeResult
	var dataMap map[string]any
	if nodeActivationEvent.InputData != nil {
		if converted, ok := nodeActivationEvent.InputData.(map[string]any); ok {
			dataMap = converted
		} else {
			dataMap = map[string]any{"value": nodeActivationEvent.InputData}
		}
	} else {
		dataMap = map[string]any{}
	}

	inputResult := models.NodeResult{
		NodeID:    nodeActivationEvent.SourceNode,
		Data:      dataMap,
		Status:    string(models.NodeStatusSuccess),
		Timestamp: time.Now(),
	}

	inputState, isReady, err := w.inputCoordinator.AddInput(
		ctx,
		nodeActivationEvent.NodeID,
		nodeActivationEvent.ExecutionID,
		nodeExecutionID,
		nodeActivationEvent.InputPort,
		inputResult,
		requirements,
	)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to add input to coordination state", "error", err)

		return w.publishNodeCompletionEvent(ctx, nodeActivationEvent, nil, err)
	}

	// 5. Only execute if node is ready
	if !isReady {
		logger.InfoContext(ctx, "Node not ready, waiting for more inputs",
			"node_execution_id", nodeExecutionID,
			"received_ports", len(inputState.ReceivedInputs),
			"required_ports", requirements.RequiredPorts,
		)

		return nil // Successfully handled, just waiting
	}

	// 6. Node is ready - get execution context and execute
	execCtx, err := w.persistence.ExecutionContextRepository().GetExecutionContext(ctx, nodeActivationEvent.ExecutionID)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to get execution context", "error", err)

		return w.publishNodeCompletionEvent(ctx, nodeActivationEvent, nil, err)
	}

	if execCtx == nil {
		err = fmt.Errorf("execution context not found: %s", nodeActivationEvent.ExecutionID)
		logger.ErrorContext(ctx, "Execution context not found", "error", err)

		return w.publishNodeCompletionEvent(ctx, nodeActivationEvent, nil, err)
	}

	// 7. Execute node with all collected inputs
	outputs, err := w.executeNodeWithInputs(ctx, node, inputState.ReceivedInputs, execCtx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to execute node", "error", err)

		return w.publishNodeCompletionEvent(ctx, nodeActivationEvent, nil, err)
	}

	logger.InfoContext(ctx, "Node executed successfully", "node_execution_id", nodeExecutionID, "output_ports", len(outputs))

	// 8. Store results in execution context
	for port, result := range outputs {
		logger.DebugContext(ctx, "Node output result", "node_id", nodeActivationEvent.NodeID, "port", port, "result", result)
		execCtx.NodeResults[nodeActivationEvent.NodeID+"::"+port] = result
	}

	// Update execution context in persistence
	err = w.persistence.ExecutionContextRepository().UpdateExecutionContext(ctx, execCtx)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to update execution context", "error", err)

		return w.publishNodeCompletionEvent(ctx, nodeActivationEvent, outputs, err)
	}

	// 9. Clean up input state after successful execution
	if err := w.inputCoordinator.CleanupNodeExecution(ctx, nodeExecutionID); err != nil {
		logger.WarnContext(ctx, "Failed to cleanup input state", "error", err)
	}

	// 10. Activate next nodes
	err = w.activateNextNodes(ctx, nodeActivationEvent.PublishedWorkflowID, nodeActivationEvent.ExecutionID, nodeActivationEvent.NodeID, outputs)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to activate next nodes", "error", err)

		return err
	}

	// 11. Publish node completion event
	return w.publishNodeCompletionEvent(ctx, nodeActivationEvent, outputs, nil)
}

// executeNodeWithInputs creates and executes a node with the provided inputs.
func (w *WorkerManager) executeNodeWithInputs(
	ctx context.Context,
	node *models.WorkflowNode,
	inputs map[string]models.NodeResult,
	execCtx *models.ExecutionContext,
) (map[string]models.NodeResult, error) {
	// Create node instance using registry
	nodeInstance, err := w.registry.CreateNode(ctx, node.Type, node.ID, node.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to create node instance: %w", err)
	}

	// Execute the node with collected inputs
	outputs, err := nodeInstance.Execute(*execCtx, inputs)
	if err != nil {
		return nil, fmt.Errorf("node execution failed: %w", err)
	}

	return outputs, nil
}

// activateNextNodes queries connections and activates connected nodes - implements direct worker-to-worker coordination.
func (w *WorkerManager) activateNextNodes(ctx context.Context, publishedWorkflowID, executionID, sourceNodeID string, outputs map[string]models.NodeResult) error {
	// Get all connections from this node
	connections, err := w.persistence.ConnectionRepository().GetConnectionsFromPublishedWorkflow(ctx, publishedWorkflowID, sourceNodeID)
	if err != nil {
		return fmt.Errorf("failed to get connections for node %s: %w", sourceNodeID, err)
	}

	w.logger.InfoContext(ctx, "Found connections to activate",
		"source_node", sourceNodeID,
		"connections_count", len(connections),
	)

	// For each connection, check if we have output on the required port and activate target node
	for _, conn := range connections {
		// Parse port IDs to extract node and port names
		_, sourcePortName, sourceOK := models.ParsePortID(conn.SourcePort)
		if !sourceOK {
			w.logger.WarnContext(ctx, "Invalid source port ID format", "port_id", conn.SourcePort)

			continue
		}

		targetNodeID, targetPortName, targetOK := models.ParsePortID(conn.TargetPort)
		if !targetOK {
			w.logger.WarnContext(ctx, "Invalid target port ID format", "port_id", conn.TargetPort)

			continue
		}

		if output, hasOutput := outputs[sourcePortName]; hasOutput {
			// Prepare activation event for target node
			activationEvent := &events.NodeActivation{
				BaseEvent: events.BaseEvent{
					ID:        fmt.Sprintf("node-activation-%d", time.Now().UnixNano()),
					Timestamp: time.Now(),
				},
				ExecutionID:         executionID,
				NodeID:              targetNodeID,
				PublishedWorkflowID: publishedWorkflowID,
				InputPort:           targetPortName,
				InputData:           output.Data,
				SourceNode:          sourceNodeID,
				SourcePort:          sourcePortName,
			}

			// Publish activation event - this implements direct worker-to-worker coordination via Kafka
			eventKey := activationEvent.NodeID + ":" + activationEvent.ExecutionID

			err = w.eventBus.Publish(ctx, eventKey, activationEvent)
			if err != nil {
				w.logger.ErrorContext(ctx, "Failed to activate next node",
					"target_node", targetNodeID,
					"source_port", sourcePortName,
					"target_port", targetPortName,
					"error", err)

				continue
			}

			w.logger.InfoContext(ctx, "Activated next node",
				"target_node", targetNodeID,
				"source_port", sourcePortName,
				"target_port", targetPortName)
		}
	}

	return nil
}

// publishNodeCompletionEvent publishes a node completion event for workflow orchestration.
func (w *WorkerManager) publishNodeCompletionEvent(ctx context.Context, nodeActivation *events.NodeActivation, outputs map[string]models.NodeResult, execError error) error {
	status := models.NodeStatusSuccess

	var errorMsg string

	if execError != nil {
		status = models.NodeStatusError
		errorMsg = execError.Error()
	}

	completionEvent := &events.NodeCompletion{
		BaseEvent: events.BaseEvent{
			ID:        fmt.Sprintf("node-completion-%d", time.Now().UnixNano()),
			Timestamp: time.Now(),
		},
		PublishedWorkflowID: nodeActivation.PublishedWorkflowID,
		ExecutionID:         nodeActivation.ExecutionID,
		NodeID:              nodeActivation.NodeID,
		Status:              status,
		OutputData:          convertNodeResultsToOutputData(outputs),
		ErrorMessage:        errorMsg,
		CompletedAt:         time.Now(),
	}

	eventKey := completionEvent.NodeID + ":" + completionEvent.ExecutionID

	return w.eventBus.Publish(ctx, eventKey, completionEvent)
}

// convertNodeResultsToOutputData converts node outputs to event data format.
func convertNodeResultsToOutputData(outputs map[string]models.NodeResult) map[string]any {
	result := make(map[string]any)
	for port, nodeResult := range outputs {
		result[port] = nodeResult.Data
	}

	return result
}

// getNodeInputRequirements gets input requirements for a node by creating an instance.
// This allows nodes to declare their coordination needs via the NodeInputRequirements interface.
func (w *WorkerManager) getNodeInputRequirements(ctx context.Context, node *models.WorkflowNode) models.InputRequirements {
	// Create the node instance (lightweight operation for purely functional nodes)
	nodeImpl, err := w.registry.CreateNode(ctx, node.Type, node.ID, node.Config)
	if err != nil {
		w.logger.WarnContext(ctx, "Could not create node for requirements, using defaults",
			"node_type", node.Type, "error", err)

		return models.DefaultInputRequirements()
	}

	// Get requirements from the node itself
	if reqNode, ok := nodeImpl.(models.NodeInputRequirements); ok {
		return reqNode.InputRequirements()
	}

	return models.DefaultInputRequirements()
}
