package controller

import (
	"STfreApi/common"
	"STfreApi/pkg/ionet"
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func getIoAPIKey(c *gin.Context) (string, bool) {
	common.OptionLock.RLock()
	enabled := common.OptionMap["model_deployment.ionet.enabled"] == "true"
	apiKey := strings.TrimSpace(common.OptionMap["model_deployment.ionet.api_key"])
	common.OptionLock.RUnlock()
	if !enabled || apiKey == "" {
		common.Fail(c, common.CodeParamError, "io.net model deployment is not enabled or api key missing")
		return "", false
	}
	return apiKey, true
}

func getIoClient(c *gin.Context) (*ionet.Client, bool) {
	apiKey, ok := getIoAPIKey(c)
	if !ok {
		return nil, false
	}
	return ionet.NewClient(apiKey), true
}

func getIoEnterpriseClient(c *gin.Context) (*ionet.Client, bool) {
	apiKey, ok := getIoAPIKey(c)
	if !ok {
		return nil, false
	}
	return ionet.NewEnterpriseClient(apiKey), true
}

func GetModelDeploymentSettings(c *gin.Context) {
	common.OptionLock.RLock()
	enabled := common.OptionMap["model_deployment.ionet.enabled"] == "true"
	hasAPIKey := strings.TrimSpace(common.OptionMap["model_deployment.ionet.api_key"]) != ""
	common.OptionLock.RUnlock()
	common.OK(c, gin.H{
		"provider":    "io.net",
		"enabled":     enabled,
		"configured":  hasAPIKey,
		"can_connect": enabled && hasAPIKey,
	})
}

func TestIoNetConnection(c *gin.Context) {
	var req struct {
		APIKey string `json:"api_key"`
	}

	rawBody, err := c.GetRawData()
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	if len(bytes.TrimSpace(rawBody)) > 0 {
		if err := json.Unmarshal(rawBody, &req); err != nil {
			common.Fail(c, common.CodeParamError, "invalid request payload")
			return
		}
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if apiKey == "" {
		common.OptionLock.RLock()
		stored := strings.TrimSpace(common.OptionMap["model_deployment.ionet.api_key"])
		common.OptionLock.RUnlock()
		if stored == "" {
			common.Fail(c, common.CodeParamError, "api_key is required")
			return
		}
		apiKey = stored
	}

	client := ionet.NewEnterpriseClient(apiKey)
	result, err := client.GetMaxGPUsPerContainer()
	if err != nil {
		if apiErr, ok := err.(*ionet.APIError); ok {
			msg := strings.TrimSpace(apiErr.Message)
			if msg == "" {
				msg = "failed to validate api key"
			}
			common.Fail(c, common.CodeServerError, msg)
			return
		}
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	hardwareCount := 0
	totalAvailable := 0
	if result != nil {
		hardwareCount = len(result.Hardware)
		totalAvailable = result.Total
		if totalAvailable == 0 {
			for _, hw := range result.Hardware {
				totalAvailable += hw.Available
			}
		}
	}

	common.OK(c, gin.H{
		"hardware_count":  hardwareCount,
		"total_available": totalAvailable,
	})
}

func requireDeploymentID(c *gin.Context) (string, bool) {
	deploymentID := strings.TrimSpace(c.Param("id"))
	if deploymentID == "" {
		common.Fail(c, common.CodeParamError, "deployment ID is required")
		return "", false
	}
	return deploymentID, true
}

func requireContainerID(c *gin.Context) (string, bool) {
	containerID := strings.TrimSpace(c.Param("container_id"))
	if containerID == "" {
		common.Fail(c, common.CodeParamError, "container ID is required")
		return "", false
	}
	return containerID, true
}

func queryInt(c *gin.Context, key string, def int) int {
	v := strings.TrimSpace(c.Query(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func mapIoNetDeployment(d ionet.Deployment) map[string]interface{} {
	created := time.Now().Unix()
	if !d.CreatedAt.IsZero() {
		created = d.CreatedAt.Unix()
	}

	hours := d.ComputeMinutesRemaining / 60
	mins := d.ComputeMinutesRemaining % 60
	timeRemaining := "completed"
	if hours > 0 {
		timeRemaining = fmt.Sprintf("%d hour %d minutes", hours, mins)
	} else if mins > 0 {
		timeRemaining = fmt.Sprintf("%d minutes", mins)
	}

	hardwareInfo := fmt.Sprintf("%s %s x%d", d.BrandName, d.HardwareName, d.HardwareQuantity)

	return map[string]interface{}{
		"id":                        d.ID,
		"deployment_name":           d.Name,
		"container_name":            d.Name,
		"status":                    strings.ToLower(strings.TrimSpace(d.Status)),
		"type":                      "Container",
		"time_remaining":            timeRemaining,
		"time_remaining_minutes":    d.ComputeMinutesRemaining,
		"hardware_info":             hardwareInfo,
		"hardware_name":             d.HardwareName,
		"brand_name":                d.BrandName,
		"hardware_quantity":         d.HardwareQuantity,
		"completed_percent":         d.CompletedPercent,
		"compute_minutes_served":    d.ComputeMinutesServed,
		"compute_minutes_remaining": d.ComputeMinutesRemaining,
		"created_at":                created,
		"updated_at":                created,
		"model_name":                "",
		"model_version":             "",
		"instance_count":            d.HardwareQuantity,
		"resource_config": map[string]interface{}{
			"cpu":    "",
			"memory": "",
			"gpu":    strconv.Itoa(d.HardwareQuantity),
		},
		"description": "",
		"provider":    "io.net",
	}
}

func computeStatusCounts(total int, deployments []ionet.Deployment) map[string]int64 {
	counts := map[string]int64{"all": int64(total)}
	for _, status := range []string{"running", "completed", "failed", "deployment requested", "termination requested", "destroyed"} {
		counts[status] = 0
	}
	for _, d := range deployments {
		status := strings.ToLower(strings.TrimSpace(d.Status))
		counts[status] = counts[status] + 1
	}
	return counts
}

func GetAllDeployments(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	dl, err := client.ListDeployments(&ionet.ListDeploymentsOptions{
		Status:    strings.ToLower(strings.TrimSpace(c.Query("status"))),
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	items := make([]map[string]interface{}, 0, len(dl.Deployments))
	for _, d := range dl.Deployments {
		items = append(items, mapIoNetDeployment(d))
	}

	common.OK(c, gin.H{
		"page":          page,
		"page_size":     pageSize,
		"total":         dl.Total,
		"items":         items,
		"status_counts": computeStatusCounts(dl.Total, dl.Deployments),
	})
}

func SearchDeployments(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	page := queryInt(c, "page", 1)
	pageSize := queryInt(c, "page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	keyword := strings.TrimSpace(c.Query("keyword"))
	dl, err := client.ListDeployments(&ionet.ListDeploymentsOptions{
		Status:    status,
		Page:      page,
		PageSize:  pageSize,
		SortBy:    "created_at",
		SortOrder: "desc",
	})
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	filtered := make([]ionet.Deployment, 0, len(dl.Deployments))
	if keyword == "" {
		filtered = dl.Deployments
	} else {
		kw := strings.ToLower(keyword)
		for _, d := range dl.Deployments {
			if strings.Contains(strings.ToLower(d.Name), kw) {
				filtered = append(filtered, d)
			}
		}
	}

	items := make([]map[string]interface{}, 0, len(filtered))
	for _, d := range filtered {
		items = append(items, mapIoNetDeployment(d))
	}

	total := dl.Total
	if keyword != "" {
		total = len(filtered)
	}

	common.OK(c, gin.H{
		"page":      page,
		"page_size": pageSize,
		"total":     total,
		"items":     items,
	})
}

func GetDeployment(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	deploymentID, ok := requireDeploymentID(c)
	if !ok {
		return
	}

	details, err := client.GetDeployment(deploymentID)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, gin.H{
		"id":              details.ID,
		"deployment_name": details.ID,
		"model_name":      "",
		"model_version":   "",
		"status":          strings.ToLower(strings.TrimSpace(details.Status)),
		"instance_count":  details.TotalContainers,
		"hardware_id":     details.HardwareID,
		"resource_config": map[string]interface{}{
			"cpu":    "",
			"memory": "",
			"gpu":    strconv.Itoa(details.TotalGPUs),
		},
		"created_at":                details.CreatedAt.Unix(),
		"updated_at":                details.CreatedAt.Unix(),
		"description":               "",
		"amount_paid":               details.AmountPaid,
		"completed_percent":         details.CompletedPercent,
		"gpus_per_container":        details.GPUsPerContainer,
		"total_gpus":                details.TotalGPUs,
		"total_containers":          details.TotalContainers,
		"hardware_name":             details.HardwareName,
		"brand_name":                details.BrandName,
		"compute_minutes_served":    details.ComputeMinutesServed,
		"compute_minutes_remaining": details.ComputeMinutesRemaining,
		"locations":                 details.Locations,
		"container_config":          details.ContainerConfig,
	})
}

func CreateDeployment(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	var req ionet.DeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	resp, err := client.DeployContainer(&req)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, gin.H{
		"deployment_id": resp.DeploymentID,
		"status":        resp.Status,
		"message":       "Deployment created successfully",
	})
}

func UpdateDeployment(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	deploymentID, ok := requireDeploymentID(c)
	if !ok {
		return
	}

	var req ionet.UpdateDeploymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	resp, err := client.UpdateDeployment(deploymentID, &req)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, gin.H{
		"status":        resp.Status,
		"deployment_id": resp.DeploymentID,
	})
}

func UpdateDeploymentName(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	deploymentID, ok := requireDeploymentID(c)
	if !ok {
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		common.Fail(c, common.CodeParamError, "deployment name cannot be empty")
		return
	}

	available, err := client.CheckClusterNameAvailability(name)
	if err != nil {
		common.Fail(c, common.CodeServerError, fmt.Sprintf("failed to check name availability: %v", err))
		return
	}
	if !available {
		common.Fail(c, common.CodeConflict, "deployment name is not available, please choose a different name")
		return
	}

	resp, err := client.UpdateClusterName(deploymentID, &ionet.UpdateClusterNameRequest{Name: name})
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, gin.H{
		"status":  resp.Status,
		"message": resp.Message,
		"id":      deploymentID,
		"name":    name,
	})
}

func ExtendDeployment(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	deploymentID, ok := requireDeploymentID(c)
	if !ok {
		return
	}

	var req ionet.ExtendDurationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	details, err := client.ExtendDeployment(deploymentID, &req)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, mapIoNetDeployment(ionet.Deployment{
		ID:                      details.ID,
		Status:                  details.Status,
		Name:                    deploymentID,
		CompletedPercent:        details.CompletedPercent,
		HardwareQuantity:        details.TotalGPUs,
		BrandName:               details.BrandName,
		HardwareName:            details.HardwareName,
		ComputeMinutesServed:    details.ComputeMinutesServed,
		ComputeMinutesRemaining: details.ComputeMinutesRemaining,
		CreatedAt:               details.CreatedAt,
	}))
}

func DeleteDeployment(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	deploymentID, ok := requireDeploymentID(c)
	if !ok {
		return
	}

	resp, err := client.DeleteDeployment(deploymentID)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, gin.H{
		"status":        resp.Status,
		"deployment_id": resp.DeploymentID,
		"message":       "Deployment termination requested successfully",
	})
}

func GetHardwareTypes(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	hardwareTypes, totalAvailable, err := client.ListHardwareTypes()
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, gin.H{
		"hardware_types":  hardwareTypes,
		"total":           len(hardwareTypes),
		"total_available": totalAvailable,
	})
}

func GetLocations(c *gin.Context) {
	client, ok := getIoClient(c)
	if !ok {
		return
	}

	locationsResp, err := client.ListLocations()
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	total := locationsResp.Total
	if total == 0 {
		total = len(locationsResp.Locations)
	}

	common.OK(c, gin.H{
		"locations": locationsResp.Locations,
		"total":     total,
	})
}

func GetAvailableReplicas(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	hardwareIDStr := strings.TrimSpace(c.Query("hardware_id"))
	if hardwareIDStr == "" {
		common.Fail(c, common.CodeParamError, "hardware_id parameter is required")
		return
	}
	hardwareID, err := strconv.Atoi(hardwareIDStr)
	if err != nil || hardwareID <= 0 {
		common.Fail(c, common.CodeParamError, "invalid hardware_id parameter")
		return
	}

	gpuCount := 1
	if gpuCountStr := strings.TrimSpace(c.Query("gpu_count")); gpuCountStr != "" {
		if parsed, err := strconv.Atoi(gpuCountStr); err == nil && parsed > 0 {
			gpuCount = parsed
		}
	}

	replicas, err := client.GetAvailableReplicas(hardwareID, gpuCount)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, replicas)
}

func GetPriceEstimation(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	var req ionet.PriceEstimationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamError, err.Error())
		return
	}

	priceResp, err := client.GetPriceEstimation(&req)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	common.OK(c, priceResp)
}

func CheckClusterNameAvailability(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	name := strings.TrimSpace(c.Query("name"))
	if name == "" {
		common.Fail(c, common.CodeParamError, "name parameter is required")
		return
	}

	available, err := client.CheckClusterNameAvailability(name)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, gin.H{"available": available, "name": name})
}

func GetDeploymentLogs(c *gin.Context) {
	client, ok := getIoClient(c)
	if !ok {
		return
	}

	deploymentID, ok := requireDeploymentID(c)
	if !ok {
		return
	}

	containerID := strings.TrimSpace(c.Query("container_id"))
	if containerID == "" {
		common.Fail(c, common.CodeParamError, "container_id parameter is required")
		return
	}

	limit := 100
	if s := strings.TrimSpace(c.Query("limit")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
			if limit > 1000 {
				limit = 1000
			}
		}
	}

	opts := &ionet.GetLogsOptions{
		Level:  c.Query("level"),
		Stream: c.Query("stream"),
		Limit:  limit,
		Cursor: c.Query("cursor"),
		Follow: c.Query("follow") == "true",
	}

	if startTime := strings.TrimSpace(c.Query("start_time")); startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			opts.StartTime = &t
		}
	}
	if endTime := strings.TrimSpace(c.Query("end_time")); endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			opts.EndTime = &t
		}
	}

	rawLogs, err := client.GetContainerLogsRaw(deploymentID, containerID, opts)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	common.OK(c, rawLogs)
}

func ListDeploymentContainers(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	deploymentID, ok := requireDeploymentID(c)
	if !ok {
		return
	}

	containers, err := client.ListContainers(deploymentID)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}

	items := make([]map[string]interface{}, 0)
	if containers != nil {
		items = make([]map[string]interface{}, 0, len(containers.Workers))
		for _, ctr := range containers.Workers {
			events := make([]map[string]interface{}, 0, len(ctr.ContainerEvents))
			for _, event := range ctr.ContainerEvents {
				events = append(events, map[string]interface{}{
					"time":    event.Time.Unix(),
					"message": event.Message,
				})
			}
			items = append(items, map[string]interface{}{
				"container_id":       ctr.ContainerID,
				"device_id":          ctr.DeviceID,
				"status":             strings.ToLower(strings.TrimSpace(ctr.Status)),
				"hardware":           ctr.Hardware,
				"brand_name":         ctr.BrandName,
				"created_at":         ctr.CreatedAt.Unix(),
				"uptime_percent":     ctr.UptimePercent,
				"gpus_per_container": ctr.GPUsPerContainer,
				"public_url":         ctr.PublicURL,
				"events":             events,
			})
		}
	}

	resp := gin.H{"total": 0, "containers": items}
	if containers != nil {
		resp["total"] = containers.Total
	}
	common.OK(c, resp)
}

func GetContainerDetails(c *gin.Context) {
	client, ok := getIoEnterpriseClient(c)
	if !ok {
		return
	}

	deploymentID, ok := requireDeploymentID(c)
	if !ok {
		return
	}
	containerID, ok := requireContainerID(c)
	if !ok {
		return
	}

	details, err := client.GetContainerDetails(deploymentID, containerID)
	if err != nil {
		common.Fail(c, common.CodeServerError, err.Error())
		return
	}
	if details == nil {
		common.Fail(c, common.CodeNotFound, "container details not found")
		return
	}

	events := make([]map[string]interface{}, 0, len(details.ContainerEvents))
	for _, event := range details.ContainerEvents {
		events = append(events, map[string]interface{}{
			"time":    event.Time.Unix(),
			"message": event.Message,
		})
	}

	common.OK(c, gin.H{
		"deployment_id":      deploymentID,
		"container_id":       details.ContainerID,
		"device_id":          details.DeviceID,
		"status":             strings.ToLower(strings.TrimSpace(details.Status)),
		"hardware":           details.Hardware,
		"brand_name":         details.BrandName,
		"created_at":         details.CreatedAt.Unix(),
		"uptime_percent":     details.UptimePercent,
		"gpus_per_container": details.GPUsPerContainer,
		"public_url":         details.PublicURL,
		"events":             events,
	})
}
