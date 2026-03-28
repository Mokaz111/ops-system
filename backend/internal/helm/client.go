package helm

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
)

// ReleaseStatus Helm release 简要状态（对应 helm status）。
type ReleaseStatus struct {
	Name      string
	Namespace string
	Status    string
	Revision  int
	Chart     string
	Version   string
}

// Client Helm 操作封装（基于 helm.sh/helm/v3/pkg/action）。
type Client struct {
	settings *cli.EnvSettings
}

// NewClient 使用 kubeconfig 路径初始化；kubeconfig 为空时使用 $KUBECONFIG / 默认路径。
func NewClient(kubeconfig string) (*Client, error) {
	s := cli.New()
	if kubeconfig != "" {
		s.KubeConfig = kubeconfig
	}
	return &Client{settings: s}, nil
}

func (c *Client) debugLog(format string, v ...interface{}) {
	if c.settings.Debug {
		fmt.Fprintf(os.Stderr, format, v...)
	}
}

func (c *Client) newActionConfig(namespace string) (*action.Configuration, error) {
	cfg := new(action.Configuration)
	driver := os.Getenv("HELM_DRIVER")
	if driver == "" {
		driver = "secrets"
	}
	if err := cfg.Init(c.settings.RESTClientGetter(), namespace, driver, c.debugLog); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Client) loadChart(chartRef string) (*chart.Chart, error) {
	cp := action.ChartPathOptions{}
	path, err := cp.LocateChart(chartRef, c.settings)
	if err != nil {
		return nil, err
	}
	return loader.Load(path)
}

// InstallRelease helm install。
func (c *Client) InstallRelease(ctx context.Context, name, chartRef, namespace string, values map[string]interface{}) error {
	if values == nil {
		values = map[string]interface{}{}
	}
	ac, err := c.newActionConfig(namespace)
	if err != nil {
		return err
	}
	ch, err := c.loadChart(chartRef)
	if err != nil {
		return err
	}
	install := action.NewInstall(ac)
	install.ReleaseName = name
	install.Namespace = namespace
	install.CreateNamespace = true
	install.Timeout = 15 * time.Minute
	install.Wait = false

	_, err = install.RunWithContext(ctx, ch, values)
	return err
}

// UpgradeRelease helm upgrade（release 已存在）。
func (c *Client) UpgradeRelease(ctx context.Context, name, chartRef, namespace string, values map[string]interface{}) error {
	if values == nil {
		values = map[string]interface{}{}
	}
	ac, err := c.newActionConfig(namespace)
	if err != nil {
		return err
	}
	ch, err := c.loadChart(chartRef)
	if err != nil {
		return err
	}
	up := action.NewUpgrade(ac)
	up.Namespace = namespace
	up.Timeout = 15 * time.Minute
	up.Wait = false

	_, err = up.RunWithContext(ctx, name, ch, values)
	return err
}

// UninstallRelease helm uninstall。
func (c *Client) UninstallRelease(ctx context.Context, name, namespace string) error {
	_ = ctx
	ac, err := c.newActionConfig(namespace)
	if err != nil {
		return err
	}
	un := action.NewUninstall(ac)
	un.Timeout = 10 * time.Minute
	un.IgnoreNotFound = true
	_, err = un.Run(name)
	return err
}

// GetReleaseStatus helm status。
func (c *Client) GetReleaseStatus(ctx context.Context, name, namespace string) (*ReleaseStatus, error) {
	_ = ctx
	ac, err := c.newActionConfig(namespace)
	if err != nil {
		return nil, err
	}
	st := action.NewStatus(ac)
	rel, err := st.Run(name)
	if err != nil {
		return nil, err
	}
	out := &ReleaseStatus{
		Name:      rel.Name,
		Namespace: rel.Namespace,
		Revision:  rel.Version,
	}
	if rel.Info != nil {
		out.Status = rel.Info.Status.String()
	}
	if rel.Chart != nil && rel.Chart.Metadata != nil {
		out.Chart = rel.Chart.Metadata.Name
		out.Version = rel.Chart.Metadata.Version
	}
	return out, nil
}

// ReleaseExists 若 release 存在则返回 true（通过 status 探测）。
func (c *Client) ReleaseExists(ctx context.Context, name, namespace string) (bool, error) {
	_, err := c.GetReleaseStatus(ctx, name, namespace)
	if err != nil {
		if isReleaseNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func isReleaseNotFound(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "not found")
}

// InstallOrUpgrade 若 release 不存在则 install，否则 upgrade。
func (c *Client) InstallOrUpgrade(ctx context.Context, name, chartRef, namespace string, values map[string]interface{}) error {
	exists, err := c.ReleaseExists(ctx, name, namespace)
	if err != nil {
		return err
	}
	if exists {
		return c.UpgradeRelease(ctx, name, chartRef, namespace, values)
	}
	return c.InstallRelease(ctx, name, chartRef, namespace, values)
}
