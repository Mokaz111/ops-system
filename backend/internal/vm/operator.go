package vm

// VMOperatorClient 占位：后续接入 controller-runtime + VictoriaMetrics Operator（VMAuth CRD 等）。
// T2.3.1 初始化与 CR 操作在集群就绪后实现。
type VMOperatorClient struct{}

// NewVMOperatorClient 预留：从 rest.Config 构建 controller-runtime client。
func NewVMOperatorClient() *VMOperatorClient {
	return &VMOperatorClient{}
}
