package service

import (
	"context"
	"errors"
	"testing"

	"ops-system/backend/internal/helm"
	"ops-system/backend/internal/model"
	"ops-system/backend/internal/repository"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ---------- 测试 fake ----------

type fakeK8s struct {
	// Get 行为
	getReturn *unstructured.Unstructured
	getErr    error

	// Scale 行为
	scaleErr error

	// PVC 行为
	resizeErr error

	// 调用记录
	getCalls   []getCRArgs
	patchCalls []patchCRArgs
	scaleCalls []scaleDeployArgs
	pvcCalls   []pvcArgs

	// PatchCustomResourceSpec 返回
	patchErr error
}

type getCRArgs struct {
	group, version, resource, namespace, name string
}
type patchCRArgs struct {
	group, version, resource, namespace, name string
	spec                                      map[string]interface{}
}
type scaleDeployArgs struct {
	namespace, name string
	replicas        int32
}
type pvcArgs struct {
	namespace, name, size string
}

func (f *fakeK8s) ScaleDeployment(_ context.Context, namespace, name string, replicas int32) error {
	f.scaleCalls = append(f.scaleCalls, scaleDeployArgs{namespace, name, replicas})
	return f.scaleErr
}

func (f *fakeK8s) GetCustomResource(_ context.Context, group, version, resource, namespace, name string) (*unstructured.Unstructured, error) {
	f.getCalls = append(f.getCalls, getCRArgs{group, version, resource, namespace, name})
	return f.getReturn, f.getErr
}

func (f *fakeK8s) PatchCustomResourceSpec(_ context.Context, group, version, resource, namespace, name string, spec map[string]interface{}) error {
	f.patchCalls = append(f.patchCalls, patchCRArgs{group, version, resource, namespace, name, spec})
	return f.patchErr
}

func (f *fakeK8s) ResizePVC(_ context.Context, namespace, name, newSize string) error {
	f.pvcCalls = append(f.pvcCalls, pvcArgs{namespace, name, newSize})
	return f.resizeErr
}

type fakeHelm struct {
	statusReturn *helm.ReleaseStatus
	statusErr    error
	upgradeErr   error

	statusCalls  int
	upgradeCalls int
}

func (f *fakeHelm) GetReleaseStatus(_ context.Context, _, _ string) (*helm.ReleaseStatus, error) {
	f.statusCalls++
	return f.statusReturn, f.statusErr
}
func (f *fakeHelm) UpgradeRelease(_ context.Context, _, _, _ string, _ map[string]interface{}) error {
	f.upgradeCalls++
	return f.upgradeErr
}

type fakeEventRepo struct {
	created []model.ScaleEvent
	listErr error
}

func (f *fakeEventRepo) Create(_ context.Context, e *model.ScaleEvent) error {
	f.created = append(f.created, *e)
	return nil
}
func (f *fakeEventRepo) List(_ context.Context, _ repository.ScaleEventListFilter) ([]model.ScaleEvent, int64, error) {
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
	return f.created, int64(len(f.created)), nil
}

// ---------- 辅助 ----------

func newUnstructured(kind, ns, name string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetKind(kind)
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetManagedFields([]metav1.ManagedFieldsEntry{})
	return u
}

func metricsInstance() *model.Instance {
	return &model.Instance{
		ID:           uuid.New(),
		InstanceName: "vm-single-a",
		ReleaseName:  "vm-single-a",
		Namespace:    "ops",
		InstanceType: "metrics",
		TemplateType: "dedicated_single",
	}
}

// ---------- 纯函数 ----------

func TestVmCRResourceFor(t *testing.T) {
	cases := map[string]string{
		"metrics": "vmsingles",
		"logs":    "vlsingles",
		"visual":  "",
		"":        "",
	}
	for in, want := range cases {
		if got := vmCRResourceFor(in); got != want {
			t.Errorf("vmCRResourceFor(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestValidateScalePolicy(t *testing.T) {
	t.Run("shared_rejected", func(t *testing.T) {
		err := validateScalePolicy(&model.Instance{TemplateType: "shared"}, &ScaleRequest{ScaleType: "vertical"})
		if !errors.Is(err, ErrScaleManagedByPlatform) {
			t.Fatalf("want ErrScaleManagedByPlatform, got %v", err)
		}
	})
	t.Run("dedicated_cluster_rejected", func(t *testing.T) {
		err := validateScalePolicy(&model.Instance{TemplateType: "dedicated_cluster"}, &ScaleRequest{ScaleType: "vertical"})
		if !errors.Is(err, ErrScaleManagedByPlatform) {
			t.Fatalf("want ErrScaleManagedByPlatform, got %v", err)
		}
	})
	t.Run("dedicated_single_allowed", func(t *testing.T) {
		if err := validateScalePolicy(&model.Instance{TemplateType: "dedicated_single"}, &ScaleRequest{ScaleType: "vertical"}); err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
	})
}

// ---------- tryPatchVMCRResources ----------

func TestTryPatchVMCRResources_CRExists_PatchesAndSkipsHelm(t *testing.T) {
	k := &fakeK8s{getReturn: newUnstructured("VMSingle", "ops", "vm-single-a")}
	h := &fakeHelm{}
	s := newScaleServiceForTest(h, k, nil, &fakeEventRepo{}, nil)

	patched, err := s.tryPatchVMCRResources(context.Background(), metricsInstance(), &ScaleRequest{CPU: "500m", Memory: "1Gi"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !patched {
		t.Fatal("want patched=true when CR exists")
	}
	if len(k.patchCalls) != 1 {
		t.Fatalf("want 1 PatchCustomResourceSpec call, got %d", len(k.patchCalls))
	}
	call := k.patchCalls[0]
	if call.resource != "vmsingles" || call.namespace != "ops" || call.name != "vm-single-a" {
		t.Errorf("patch target wrong: %+v", call)
	}
	res, ok := call.spec["resources"].(map[string]interface{})
	if !ok {
		t.Fatalf("spec.resources missing: %+v", call.spec)
	}
	limits, _ := res["limits"].(map[string]interface{})
	if limits["cpu"] != "500m" || limits["memory"] != "1Gi" {
		t.Errorf("unexpected limits: %+v", limits)
	}
	if h.statusCalls != 0 || h.upgradeCalls != 0 {
		t.Error("helm should not be touched when CR patch succeeds")
	}
}

func TestTryPatchVMCRResources_CRNotFound_ReturnsFalse(t *testing.T) {
	k := &fakeK8s{getReturn: nil}
	s := newScaleServiceForTest(nil, k, nil, &fakeEventRepo{}, nil)

	patched, err := s.tryPatchVMCRResources(context.Background(), metricsInstance(), &ScaleRequest{CPU: "500m"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if patched {
		t.Fatal("want patched=false when CR missing")
	}
	if len(k.patchCalls) != 0 {
		t.Error("patch should not be called when CR missing")
	}
}

func TestTryPatchVMCRResources_NonVMType(t *testing.T) {
	k := &fakeK8s{getReturn: newUnstructured("Deployment", "ops", "x")}
	s := newScaleServiceForTest(nil, k, nil, &fakeEventRepo{}, nil)

	inst := metricsInstance()
	inst.InstanceType = "visual"

	patched, err := s.tryPatchVMCRResources(context.Background(), inst, &ScaleRequest{CPU: "1"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if patched {
		t.Fatal("visual instance must not be CR-patched")
	}
	if len(k.getCalls) != 0 {
		t.Error("get should not be called for non-vm instance")
	}
}

func TestTryPatchVMCRResources_GetError_Propagates(t *testing.T) {
	k := &fakeK8s{getErr: errors.New("api down")}
	s := newScaleServiceForTest(nil, k, nil, &fakeEventRepo{}, nil)

	patched, err := s.tryPatchVMCRResources(context.Background(), metricsInstance(), &ScaleRequest{CPU: "1"})
	if err == nil {
		t.Fatal("want error when Get fails")
	}
	if patched {
		t.Fatal("want patched=false on error")
	}
}

// ---------- tryPatchVMCRStorage ----------

func TestTryPatchVMCRStorage_CRExists_PatchesAndSkipsPVC(t *testing.T) {
	k := &fakeK8s{getReturn: newUnstructured("VMSingle", "ops", "vm-single-a")}
	s := newScaleServiceForTest(nil, k, nil, &fakeEventRepo{}, nil)

	patched, err := s.tryPatchVMCRStorage(context.Background(), metricsInstance(), &ScaleRequest{Storage: "100Gi"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !patched {
		t.Fatal("want patched=true")
	}
	if len(k.patchCalls) != 1 {
		t.Fatalf("want 1 patch call, got %d", len(k.patchCalls))
	}
	spec := k.patchCalls[0].spec
	storage, _ := spec["storage"].(map[string]interface{})
	res, _ := storage["resources"].(map[string]interface{})
	req, _ := res["requests"].(map[string]interface{})
	if req["storage"] != "100Gi" {
		t.Errorf("expected spec.storage.resources.requests.storage=100Gi, got %v", req["storage"])
	}
	if len(k.pvcCalls) != 0 {
		t.Error("PVC resize should not be invoked")
	}
}

func TestTryPatchVMCRStorage_CRNotFound(t *testing.T) {
	k := &fakeK8s{getReturn: nil}
	s := newScaleServiceForTest(nil, k, nil, &fakeEventRepo{}, nil)

	patched, err := s.tryPatchVMCRStorage(context.Background(), metricsInstance(), &ScaleRequest{Storage: "100Gi"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if patched {
		t.Fatal("want patched=false when CR missing")
	}
	if len(k.patchCalls) != 0 {
		t.Error("patch should not happen when CR missing")
	}
}

// ---------- recordEvent ----------

func TestRecordEvent_SuccessAndFailure(t *testing.T) {
	repo := &fakeEventRepo{}
	s := newScaleServiceForTest(nil, nil, nil, repo, nil)

	inst := metricsInstance()
	s.recordEvent(context.Background(), inst, &ScaleRequest{ScaleType: "vertical", CPU: "1", Operator: "alice"}, "cr_patch", nil)
	s.recordEvent(context.Background(), inst, &ScaleRequest{ScaleType: "vertical", CPU: "1", Operator: "alice"}, "helm_upgrade", errors.New("boom"))

	if len(repo.created) != 2 {
		t.Fatalf("want 2 events, got %d", len(repo.created))
	}
	if repo.created[0].Status != "success" || repo.created[0].Method != "cr_patch" {
		t.Errorf("first event wrong: %+v", repo.created[0])
	}
	if repo.created[1].Status != "failed" || repo.created[1].Method != "helm_upgrade" || repo.created[1].ErrorMessage != "boom" {
		t.Errorf("second event wrong: %+v", repo.created[1])
	}
}

func TestRecordEvent_NilRepoIsSafe(t *testing.T) {
	s := newScaleServiceForTest(nil, nil, nil, nil, nil)
	// 不应 panic
	s.recordEvent(context.Background(), metricsInstance(), &ScaleRequest{ScaleType: "vertical"}, "cr_patch", nil)
}

// ---------- ListScaleEvents ----------

func TestListScaleEvents_InvalidPagination(t *testing.T) {
	repo := &fakeEventRepo{}
	s := newScaleServiceForTest(nil, nil, nil, repo, nil)

	_, _, err := s.ListScaleEvents(context.Background(), repository.ScaleEventListFilter{}, 1, 0)
	if !errors.Is(err, ErrInvalidPagination) {
		t.Fatalf("want ErrInvalidPagination, got %v", err)
	}
	_, _, err = s.ListScaleEvents(context.Background(), repository.ScaleEventListFilter{}, 1, 500)
	if !errors.Is(err, ErrInvalidPagination) {
		t.Fatalf("want ErrInvalidPagination for oversize, got %v", err)
	}
}

func TestListScaleEvents_NilRepoReturnsEmpty(t *testing.T) {
	s := newScaleServiceForTest(nil, nil, nil, nil, nil)
	list, total, err := s.ListScaleEvents(context.Background(), repository.ScaleEventListFilter{}, 1, 20)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if list != nil || total != 0 {
		t.Errorf("want (nil, 0), got (%v, %d)", list, total)
	}
}
