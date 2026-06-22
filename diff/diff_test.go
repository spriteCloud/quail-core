package diff

import "testing"

const sample = `diff --git a/src/foo.ts b/src/foo.ts
index 0000000..1111111 100644
--- a/src/foo.ts
+++ b/src/foo.ts
@@ -1,3 +1,6 @@
 export function existing() {}
+
+export function added(a: number, b: number) {
+  return a + b
+}
diff --git a/src/bar.ts b/src/bar.ts
new file mode 100644
--- /dev/null
+++ b/src/bar.ts
@@ -0,0 +1,2 @@
+export const X = 1
+export const Y = 2
`

func TestParse(t *testing.T) {
	files := Parse(sample)
	if len(files) != 2 {
		t.Fatalf("want 2 files, got %d", len(files))
	}
	if files[0].Path != "src/foo.ts" {
		t.Errorf("foo path: %s", files[0].Path)
	}
	if !Intersects(files[0].Added, 3, 6) {
		t.Errorf("foo should report added in 3..6: %+v", files[0].Added)
	}
	if Intersects(files[0].Added, 1, 1) {
		t.Errorf("line 1 was context, not added")
	}
	if files[1].Status != "added" {
		t.Errorf("bar status: %s", files[1].Status)
	}
	if !Intersects(files[1].Added, 1, 2) {
		t.Errorf("bar should report added 1..2: %+v", files[1].Added)
	}
}
