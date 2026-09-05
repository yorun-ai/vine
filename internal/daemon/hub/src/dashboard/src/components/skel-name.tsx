export function SkelName({
  skelName,
  domain = skelName.slice(0, Math.max(0, skelName.lastIndexOf('.'))),
}: {
  skelName: string
  domain?: string
}) {
  return (
    <>
      <span className="text-primary">{domain}</span>
      {skelName.slice(domain.length)}
    </>
  )
}
