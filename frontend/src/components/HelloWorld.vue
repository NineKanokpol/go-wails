<script setup>
import { onMounted, ref } from "vue";
import { CheckUpdate, ApplyUpdate, GetButtonText, SetButtonText, Quit,GetAppVersion } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime"; // ใช้ event จาก Go → JS ได้ :contentReference[oaicite:4]{index=4}

const manifestURL = ref("https://github.com/NineKanokpol/go-wails/releases/latest/download/manifest.json");

const btnText = ref("Output");
const status = ref("");
const output = ref("");
const updateInfo = ref(null);
const currentVersion = ref("");

async function loadButtonText() {
  btnText.value = await GetButtonText(); // เปิดแอปใหม่ก็ยังจำได้ เพราะ Go load state ไว้แล้ว
}

async function checkUpdateOnStartup() {
  // ✅ กัน cache ด้วยการใส่ timestamp
  const url = `${manifestURL.value}?t=${Date.now()}`;
  updateInfo.value = await CheckUpdate(url);

  if (updateInfo.value?.error) {
    status.value = `CheckUpdate error: ${updateInfo.value.error}`;
    return;
  }

  if (updateInfo.value?.hasUpdate) {
    status.value = `พบอัปเดต: ${updateInfo.value.latest} (ของคุณ ${updateInfo.value.current})`;
    await SetButtonText("Update available");
    await loadButtonText();
  } else {
    status.value = `เป็นเวอร์ชันล่าสุดแล้ว (current=${updateInfo.value.current}, latest=${updateInfo.value.latest})`;

    // ✅ (แนะนำ) reset ปุ่มกลับเป็น Output ทุกครั้งถ้าไม่มีอัปเดต
    await SetButtonText("Output");
    await loadButtonText();
  }
}

// ปุ่ม Output / Update
async function onMainButtonClick() {
  // ถ้ามีอัปเดต → ให้กดแล้วอัปเดต
  if (updateInfo.value?.hasUpdate) {
    status.value = "กำลังอัปเดต...";
    const res = await ApplyUpdate(updateInfo.value.url, updateInfo.value.sha256);
    status.value = res.ok ? res.message : `อัปเดตล้มเหลว: ${res.error}`;

    // หลังอัปเดตสำเร็จ เปลี่ยนปุ่มเป็น Restart
    if (res.ok) {
      await SetButtonText("Restart app");
      await loadButtonText();

      await RestartApp();
    }
    return;
  }

  // ถ้าไม่มีอัปเดต → ทำหน้าที่ Output ปกติ (ตัวอย่าง “ปริ้น”)
  output.value = "Output clicked ✅ (ไม่มีอัปเดต)";
}

async function onRestart() {
  Quit(); // ให้ผู้ใช้เปิดใหม่เองแบบชัวร์สุด
}

onMounted(async () => {
  currentVersion.value = await GetAppVersion();
  await loadButtonText();
  await checkUpdateOnStartup();
  setInterval(() => {
    checkUpdateOnStartup();
  }, 5 * 60 * 1000);
  EventsOn("update:applied", async () => {
    // ถ้าต้องการทำอะไรเพิ่มหลัง apply
  });
});
</script>

<template>
  <div style="padding:20px;">
    <div style="margin-bottom:10px;">
      <b>StatusVersion 5 min:</b> {{ status }}
    </div>
    <button @click="onMainButtonClick" style="padding:10px 14px; margin-right:8px;">
      {{ btnText }}
    </button>

    <button v-if="btnText === 'Restart app'" @click="onRestart" style="padding:10px 14px;">
      Quit (แล้วเปิดใหม่)
    </button>

    <p style="margin-top:16px;">Output {{ currentVersion }}: {{ output }}</p>
  </div>
</template>