# Z.AI integration notes

المصدر الرسمي: https://docs.z.ai/guides/develop/openai/python

توثّق الصفحة أن Z.AI يوفّر واجهة متوافقة مع OpenAI، وأن عميل OpenAI يمكن تهيئته باستخدام `base_url="https://api.z.ai/api/paas/v4/"` مع مفتاح API. المثال الرسمي يستخدم `chat.completions.create` ونموذج `glm-5.3`. لذلك سيكون تكامل Go اختيارياً عبر HTTP/JSON من المكتبة القياسية، مع قراءة `ZAI_API_KEY` و`ZAI_BASE_URL` و`ZAI_MODEL` من البيئة وعدم تضمين الأسرار في المستودع.

نطاق الدفعة: عميل chat صغير قابل للحقن والاختبار، لا يربط guest CPU أو syscall path بالمزود، ولا يجعل Z.AI شرطاً لتشغيل iSH core أو CGO_ENABLED=0.


المصدر الرسمي الثاني: https://docs.z.ai/guides/develop/http/introduction

يستخدم REST header بصيغة `Authorization: Bearer YOUR_API_KEY` و`Content-Type: application/json`. مسار chat هو `POST https://api.z.ai/api/paas/v4/chat/completions`، والطلب يتضمن `model`, `messages`, ويدعم `temperature` و`max_tokens`. لن يُجرى أي طلب حقيقي ما لم يوفّر المستخدم مفتاحاً أو يفعّل موصلاً مناسباً.
