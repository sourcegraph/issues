<style>
.estimator label {
    display: flex;
}

.estimator .radioInput label {
    display: inline-flex;
    align-items: center;
    margin-left: .5rem;
}

.estimator .radioInput label span {
    margin-left: .25rem;
    margin-right: .25rem;
}

.estimator input[type=range] {
    width: 15rem;
}

.estimator .post-label {
    font-size: 16px;
    margin-left: 0.5rem;
}

.estimator .copy-as-markdown {
    width: 40rem;
    height: 8rem;
}

.estimator a[title]:hover:after {
  content: attr(title);
  background: red;
  position: relative;
  z-index: 1000;
  top: 16px;
  left: 0;
}
</style>

<script src="https://storage.googleapis.com/sourcegraph-resource-estimator/go_1_14_wasm_exec.js"></script>
<script src="https://storage.googleapis.com/sourcegraph-resource-estimator/launch_script.js?v2" version="346600f"></script>

<form id="root"></form>
