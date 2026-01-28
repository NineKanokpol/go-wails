<script setup>
import { onMounted, ref } from "vue";
import { CheckUpdate, ApplyUpdate, GetButtonText, SetButtonText, Quit } from "../../wailsjs/go/main/App";
import { EventsOn } from "../../wailsjs/runtime/runtime"; // ใช้ event จาก Go → JS ได้ :contentReference[oaicite:4]{index=4}

const manifestURL = ref("https://yourdomain.com/myapp/manifest.json");

const btnText = ref("Output");
const status = ref("");
const output = ref("");
const updateInfo = ref(null);

async function loadButtonText() {
  btnText.value = await GetButtonText(); // เปิดแอปใหม่ก็ยังจำได้ เพราะ Go load state ไว้แล้ว
}

async function checkUpdateOnStartup() {
  updateInfo.value = await CheckUpdate(manifestURL.value);

  if (updateInfo.value?.hasUpdate) {
    status.value = `พบอัปเดต: ${updateInfo.value.latest} (ของคุณ ${updateInfo.value.current})`;
    // เปลี่ยน text ปุ่ม + เซฟลงไฟล์ state
    await SetButtonText("Update available");
    await loadButtonText();
  } else {
    status.value = "เป็นเวอร์ชันล่าสุดแล้ว";
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
  await loadButtonText();
  await checkUpdateOnStartup();

  EventsOn("update:applied", async () => {
    // ถ้าต้องการทำอะไรเพิ่มหลัง apply
  });
});
</script>

<template>
  <div style="padding:20px;">
    <div style="margin-bottom:10px;">
      <b>StatusTTTTTT:</b> {{ status }}
    </div>

    <button @click="onMainButtonClick" style="padding:10px 14px; margin-right:8px;">
      {{ btnText }}
    </button>

    <button v-if="btnText === 'Restart app'" @click="onRestart" style="padding:10px 14px;">
      Quit (แล้วเปิดใหม่)
    </button>

    <p style="margin-top:16px;">Output: {{ output }}</p>
  </div>
</template>