import { useState } from 'react'
import { FileText } from 'lucide-react'
import { createI18nAdapter } from '@ztd-me/ui'
import {
  Accordion, AccordionContent, AccordionItem, AccordionTrigger,
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogTitle, AlertDialogTrigger,
  Badge, Button, Card, CardContent, CardDescription, CardHeader, CardTitle, Checkbox, ColorGrid, Collapsible, CollapsibleContent, CollapsibleTrigger,
  Drawer, DrawerClose, DrawerContent, DrawerDescription, DrawerTitle, DrawerTrigger,
  Dialog, DialogClose, DialogContent, DialogDescription, DialogTitle, DialogTrigger,
  Input, Label, ScrollArea, Select, SelectContent, SelectItem, SelectRoot, SelectTrigger, SelectValue,
  SettingSection, Switch,
  Sheet, SheetClose, SheetContent, SheetDescription, SheetTitle, SheetTrigger,
  Tabs, TabsContent, TabsList, TabsTrigger, Textarea, Tooltip, TooltipContent, TooltipProvider, TooltipTrigger, useI18n,
} from '@ztd-me/ui/client'

const resources = {
  en: { fixture: { welcome: 'Ready {{count}}' } },
  'zh-CN': { fixture: { welcome: '已就绪 {{count}}' } },
}
function InjectedLocale(): React.JSX.Element {
  const [adapter] = useState(() => createI18nAdapter({
    resources,
    instance: { translate: ({ locale, key, values }) =>
      resources[locale].fixture[key as 'welcome'].replace('{{count}}', String(values.count)) },
  }))
  const { t } = useI18n(adapter)
  return <output id="injected-locale">{t('fixture', 'welcome', { count: 1 })}</output>
}
function Fields(): React.JSX.Element {
  const [submitted, setSubmitted] = useState('')
  return (
    <form onSubmit={(event) => {
      event.preventDefault()
      setSubmitted(JSON.stringify(Object.fromEntries(new FormData(event.currentTarget))))
    }}>
      <Label htmlFor="primitive-input">Fixture name</Label>
      <Input id="primitive-input" name="name" defaultValue="Preserved" />
      <Label>Wrapped field<Input aria-label="Wrapped field" /></Label>
      <Label htmlFor="primitive-disabled">Disabled name</Label>
      <Input id="primitive-disabled" name="disabledName" defaultValue="Unchanged" disabled />
      <Button type="submit" disabled>Disabled submit</Button>
      <Label htmlFor="primitive-note">Fixture note</Label>
      <Textarea id="primitive-note" name="note" defaultValue="Draft" />
      <Label htmlFor="primitive-native">Native choice</Label>
      <Select id="primitive-native" name="choice" defaultValue="two">
        <option value="one">One</option><option value="two">Two</option>
      </Select>
      <Label><Checkbox name="accepted" defaultChecked />Accepted</Label>
      <Button type="submit">Submit fixture</Button>
      <Button type="button" variant="outline">Outline fixture</Button>
      <Button type="button" variant="ghost">Ghost fixture</Button>
      <Button type="button" variant="destructive">Destructive fixture</Button>
      <p className="fixture-destructive-notice">Destructive notice</p>
      <output id="submitted">{submitted}</output>
    </form>
  )
}
function Overlays(): React.JSX.Element {
  return (
    <>
      <Dialog>
        <DialogTrigger asChild><Button>Open dialog</Button></DialogTrigger>
        <DialogContent>
          <DialogTitle>Fixture dialog</DialogTitle><DialogDescription>Focusable content</DialogDescription>
          <Input aria-label="Dialog input" />
          <DialogClose asChild><Button variant="outline">Close dialog</Button></DialogClose>
        </DialogContent>
      </Dialog>
      <AlertDialog>
        <AlertDialogTrigger asChild><Button>Open confirmation</Button></AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogTitle>Fixture confirmation</AlertDialogTitle>
          <AlertDialogDescription>Choose an action</AlertDialogDescription>
          <AlertDialogCancel asChild><Button variant="outline">Cancel action</Button></AlertDialogCancel>
          <AlertDialogAction asChild><Button>Confirm action</Button></AlertDialogAction>
        </AlertDialogContent>
      </AlertDialog>
      <Sheet>
        <SheetTrigger asChild><Button>Open sheet</Button></SheetTrigger>
        <SheetContent>
          <SheetTitle>Fixture sheet</SheetTitle><SheetDescription>Side content</SheetDescription>
          <SheetClose asChild><Button variant="outline">Close sheet</Button></SheetClose>
        </SheetContent>
      </Sheet>
    </>
  )
}
function Selections(): React.JSX.Element {
  return (
    <>
      <Accordion type="single" collapsible>
        <AccordionItem value="details"><AccordionTrigger>Fixture details</AccordionTrigger>
          <AccordionContent>Expanded details</AccordionContent>
        </AccordionItem>
      </Accordion>
      <Collapsible><CollapsibleTrigger asChild><Button>Toggle section</Button></CollapsibleTrigger>
        <CollapsibleContent>Visible section</CollapsibleContent>
      </Collapsible>
      <Tabs defaultValue="one">
        <TabsList aria-label="Fixture tabs"><TabsTrigger value="one"><FileText aria-hidden="true" size={16} /><span>First tab</span></TabsTrigger>
          <TabsTrigger value="two">Second tab</TabsTrigger>
        </TabsList>
        <TabsContent value="one">First panel</TabsContent><TabsContent value="two">Second panel</TabsContent>
      </Tabs>
      <SelectRoot defaultValue="one">
        <SelectTrigger aria-label="Enhanced choice"><SelectValue /></SelectTrigger>
        <SelectContent><SelectItem value="one">First choice</SelectItem>
          <SelectItem value="two">Second choice</SelectItem>
        </SelectContent>
      </SelectRoot>
    </>
  )
}
function TypographyRoles(): React.JSX.Element {
  return (
    <section aria-label="Typography roles">
      <Card>
        <CardHeader>
          <CardTitle>Typography role sample</CardTitle>
          <CardDescription>Supporting copy 中文</CardDescription>
        </CardHeader>
        <CardContent>
          <p id="typography-body">Ordinary interface copy 中文</p>
          <p id="typography-reading" className="text-reading">Reading prose with preserved legibility 中文</p>
          <Button id="reading-class-override" variant="outline" className="text-reading">Reading role override</Button>
          <Button id="input-class-override" variant="outline" className="text-editable">Input role override</Button>
          <Label htmlFor="reading-input-override">Explicit reading input override</Label>
          <Input id="reading-input-override" className="text-reading" defaultValue="Consumer-owned typography" />
          <Button size="xs" variant="outline">Extra small target</Button>
          <Button size="sm" variant="outline">Small target</Button>
          <Button size="icon-xs" variant="outline" aria-label="Extra small icon target"><FileText size={16} aria-hidden="true" /></Button>
          <Button size="icon-sm" variant="outline" aria-label="Small icon target"><FileText size={16} aria-hidden="true" /></Button>
        </CardContent>
      </Card>
    </section>
  )
}
function SharedExtras(): React.JSX.Element {
  const [color, setColor] = useState('automatic')
  return (
    <>
      <SettingSection title="Settings sample" description="Shared disclosure" summary="Choose a sample"
        status={<Badge variant="secondary">Ready</Badge>}>
        <Input aria-label="Settings input" defaultValue="Preserved setting" />
      </SettingSection>
      <Label htmlFor="fixture-switch">Shared switch</Label><Switch id="fixture-switch" defaultChecked />
      <ColorGrid label="Sample colors" value={color} onValueChange={setColor}
        options={[
          { value: 'automatic', label: 'Automatic', foreground: 'var(--ztd-foreground)' },
          { value: 'accent', label: 'Accent sample', foreground: 'var(--ztd-accent)' },
          { value: 'muted', label: 'Muted sample', background: 'var(--ztd-muted)' },
        ]} />
      <output id="selected-color">{color}</output>
      <Drawer>
        <DrawerTrigger asChild><Button>Open drawer</Button></DrawerTrigger>
        <DrawerContent>
          <DrawerTitle>Fixture drawer</DrawerTitle><DrawerDescription>Shared mobile options</DrawerDescription>
          <Input aria-label="Drawer input" />
          <DrawerClose asChild><Button variant="outline">Close drawer</Button></DrawerClose>
        </DrawerContent>
      </Drawer>
      <Button className="h-8 min-h-0 p-1 text-help" variant="outline">Compact override</Button>
    </>
  )
}
export function PrimitiveFixture(): React.JSX.Element {
  return (
    <section aria-label="Primitive fixture">
      <h1>Foundation primitives</h1><InjectedLocale /><Fields /><Selections /><Overlays /><SharedExtras /><TypographyRoles />
      <ScrollArea style={{ height: 120, width: 240 }} type="always"
        viewportProps={{ tabIndex: 0, role: 'region', 'aria-label': 'Scrollable sample' }}>
        {Array.from({ length: 35 }, (_, index) => <p key={index}>Scrollable row {index + 1}</p>)}
      </ScrollArea>
      <TooltipProvider delayDuration={0}><Tooltip>
        <TooltipTrigger asChild><Button>Tooltip anchor</Button></TooltipTrigger>
        <TooltipContent>Fixture tooltip</TooltipContent>
      </Tooltip></TooltipProvider>
    </section>
  )
}
