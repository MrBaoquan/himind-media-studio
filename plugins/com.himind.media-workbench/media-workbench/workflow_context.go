package main

// workflow_context.go —— 让 Capability 能消费 Workflow 上游步骤输出。
//
// Workflow 引擎会把已完成步骤的输出放在 workflow_context.steps[STEP_ID]。
// 独立调用能力时仍使用 plan/project_id；Workflow 调用时可以传 plan_step /
// project_step，避免把大 JSON 重新塞回步骤输入。

func workflowStepOutput(context map[string]any, stepID string) any {
	if context == nil || stepID == "" {
		return nil
	}
	steps, ok := context["steps"].(map[string]any)
	if !ok {
		return nil
	}
	return steps[stepID]
}

func workflowObject(output any) map[string]any {
	object, _ := output.(map[string]any)
	return object
}

func planFromInput(in requestInput) map[string]any {
	if in.Plan != nil {
		return in.Plan
	}
	output := workflowObject(workflowStepOutput(in.WorkflowContext, in.PlanStep))
	if output == nil {
		return nil
	}
	if plan, ok := output["plan"].(map[string]any); ok {
		return plan
	}
	return output
}

func projectIDFromInput(in requestInput) string {
	if in.ProjectID != "" {
		return in.ProjectID
	}
	output := workflowObject(workflowStepOutput(in.WorkflowContext, in.ProjectStep))
	if output == nil {
		return ""
	}
	if project, ok := output["project"].(map[string]any); ok {
		if id := asString(project["id"]); id != "" {
			return id
		}
	}
	return asString(output["project_id"])
}
func dataFromInput(in requestInput) map[string]any {
	if in.Data != nil {
		return in.Data
	}
	output := workflowObject(workflowStepOutput(in.WorkflowContext, in.DataStep))
	if output == nil {
		return nil
	}
	if data, ok := output["data"].(map[string]any); ok {
		return data
	}
	if items, ok := output["items"].([]any); ok {
		return map[string]any{"items": items}
	}
	return nil
}
