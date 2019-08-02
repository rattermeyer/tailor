package openshift

import (
	"testing"
)

func TestConfigFilterByKind(t *testing.T) {
	byteList := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    name: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: bar
  data:
    bar: baz
kind: List
metadata: {}
`)

	filter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "",
	}

	list, _ := NewTemplateBasedResourceList(filter, byteList)

	if len(list.Items) != 1 {
		t.Errorf("One item should have been extracted, got %v items.", len(list.Items))
		return
	}

	item := list.Items[0]
	if item.Kind != "PersistentVolumeClaim" {
		t.Errorf("Item should have been a PersistentVolumeClaim, got %s.", item.Kind)
	}
}

func TestConfigFilterByName(t *testing.T) {
	byteList := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    name: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    name: bar
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi
    storageClassName: gp2
  status: {}
kind: List
metadata: {}
`)

	filter := &ResourceFilter{
		Kinds: []string{},
		Name:  "PersistentVolumeClaim/foo",
		Label: "",
	}

	list, _ := NewTemplateBasedResourceList(filter, byteList)

	if len(list.Items) != 1 {
		t.Errorf("One item should have been extracted, got %v items.", len(list.Items))
		return
	}

	item := list.Items[0]
	if item.Name != "foo" {
		t.Errorf("Item should have had name foo, got %s.", item.Name)
	}
}

func TestConfigFilterBySelector(t *testing.T) {
	byteList := []byte(
		`apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    labels:
      app: foo
    name: foo
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 5Gi
    storageClassName: gp2
  status: {}
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    labels:
      app: bar
    name: bar
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 1Gi
    storageClassName: gp2
  status: {}
- apiVersion: v1
  kind: ConfigMap
  metadata:
    labels:
      app: foo
    name: foo
  data:
    bar: baz
- apiVersion: v1
  kind: ConfigMap
  metadata:
    labels:
      app: bar
    name: bar
  data:
    bar: baz
- apiVersion: v1
  data:
    auth-token: abcdef
  kind: Secret
  metadata:
    name: bar
    labels:
      app: bar
  type: opaque
kind: List
metadata: {}
`)

	pvcFilter := &ResourceFilter{
		Kinds: []string{"PersistentVolumeClaim"},
		Name:  "",
		Label: "app=foo",
	}
	cmFilter := &ResourceFilter{
		Kinds: []string{"ConfigMap"},
		Name:  "",
		Label: "app=foo",
	}
	secretFilter := &ResourceFilter{
		Kinds: []string{"Secret"},
		Name:  "",
		Label: "app=foo",
	}

	pvcList, _ := NewTemplateBasedResourceList(pvcFilter, byteList)

	if len(pvcList.Items) != 1 {
		t.Errorf("One item should have been extracted, got %v items.", len(pvcList.Items))
	}

	_, err := pvcList.getItem("PersistentVolumeClaim", "foo")
	if err != nil {
		t.Errorf("Item foo should have been present.")
	}

	cmList, _ := NewTemplateBasedResourceList(cmFilter, byteList)

	if len(cmList.Items) != 1 {
		t.Errorf("One item should have been extracted, got %v items.", len(cmList.Items))
	}

	_, err = cmList.getItem("ConfigMap", "foo")
	if err != nil {
		t.Errorf("Item should have been present.")
	}

	secretList, _ := NewTemplateBasedResourceList(secretFilter, byteList)

	if len(secretList.Items) != 0 {
		t.Errorf("No item should have been extracted, got %v items.", len(secretList.Items))
	}
}
